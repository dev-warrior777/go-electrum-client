// Copyright (C) 2015-2016 The Lightning Network Developers

package wltbtc

import (
	"errors"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/coinset"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/btcutil/txsort"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcwallet/wallet/txauthor"
	"github.com/btcsuite/btcwallet/wallet/txrules"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

func (w *BtcElectrumWallet) Broadcast(tx *wire.MsgTx) error {
	// Our own tx; don't keep track of false positives
	_, err := w.txstore.AddTransaction(tx, 0, time.Now())
	if err != nil {
		return err
	}

	fmt.Printf("Broadcasting tx %s to electrumX server", tx.TxHash().String())
	//TODO: move to client
	return nil
}

type Coin struct {
	TxHash       *chainhash.Hash
	TxIndex      uint32
	TxValue      btcutil.Amount
	TxNumConfs   int64
	ScriptPubKey []byte
}

func (c *Coin) Hash() *chainhash.Hash { return c.TxHash }
func (c *Coin) Index() uint32         { return c.TxIndex }
func (c *Coin) Value() btcutil.Amount { return c.TxValue }
func (c *Coin) PkScript() []byte      { return c.ScriptPubKey }
func (c *Coin) NumConfs() int64       { return c.TxNumConfs }
func (c *Coin) ValueAge() int64       { return int64(c.TxValue) * c.TxNumConfs }

func NewCoin(txHash *chainhash.Hash, index uint32, value btcutil.Amount, numConfs int64, scriptPubKey []byte) coinset.Coin {
	c := &Coin{
		TxHash:       txHash,
		TxIndex:      index,
		TxValue:      value,
		TxNumConfs:   numConfs,
		ScriptPubKey: scriptPubKey,
	}
	return coinset.Coin(c)
}

// gather
func (w *BtcElectrumWallet) gatherCoins() map[coinset.Coin]*hdkeychain.ExtendedKey {
	tip := w.blockchainTip
	utxos, _ := w.txstore.Utxos().GetAll()
	m := make(map[coinset.Coin]*hdkeychain.ExtendedKey)
	for _, u := range utxos {
		if u.WatchOnly {
			continue
		}
		var confirmations int64
		if u.AtHeight > 0 {
			confirmations = tip - u.AtHeight
		}
		c := NewCoin(&u.Op.Hash, u.Op.Index, btcutil.Amount(u.Value), confirmations, u.ScriptPubkey)
		addr, err := w.ScriptToAddress(u.ScriptPubkey)
		if err != nil {
			continue
		}
		key, err := w.keyManager.GetKeyForScript(addr.ScriptAddress())
		if err != nil {
			continue
		}
		m[c] = key
	}
	return m
}

func (w *BtcElectrumWallet) Spend(amount int64, addr btcutil.Address, feeLevel wallet.FeeLevel, referenceID string, spendAll bool) (*wire.MsgTx, error) {
	var (
		tx  *wire.MsgTx
		err error
	)
	if spendAll {
		tx, err = w.buildSpendAllTx(addr, feeLevel)
		if err != nil {
			return nil, err
		}
	} else {
		tx, err = w.buildTx(amount, addr, feeLevel, nil)
		if err != nil {
			return nil, err
		}
	}

	return tx, nil
}

func (w *BtcElectrumWallet) EstimateFee(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, feePerByte int64) int64 {
	tx := new(wire.MsgTx)
	for _, out := range outs {
		scriptPubKey, _ := txscript.PayToAddrScript(out.Address)
		output := wire.NewTxOut(out.Value, scriptPubKey)
		tx.TxOut = append(tx.TxOut, output)
	}
	estimatedSize := EstimateSerializeSize(len(ins), tx.TxOut, false, P2PKH)
	fee := estimatedSize * int(feePerByte)
	return int64(fee)
}

// Build a spend transaction for the amount and return the transaction fee
func (w *BtcElectrumWallet) EstimateSpendFee(amount int64, feeLevel wallet.FeeLevel) (uint64, error) {
	// Since this is an estimate we can use a dummy output address. Let's use a long one so we don't under estimate.
	addr, err := btcutil.DecodeAddress("bc1qxtq7ha2l5qg70atpwp3fus84fx3w0v2w4r2my7gt89ll3w0vnlgspu349h", w.params)
	if err != nil {
		return 0, err
	}
	tx, err := w.buildTx(amount, addr, feeLevel, nil)
	if err != nil {
		return 0, err
	}
	var outval int64
	for _, output := range tx.TxOut {
		outval += output.Value
	}
	var inval int64
	utxos, err := w.txstore.Utxos().GetAll()
	if err != nil {
		return 0, err
	}
	for _, input := range tx.TxIn {
		for _, utxo := range utxos {
			if utxo.Op.Hash.IsEqual(&input.PreviousOutPoint.Hash) && utxo.Op.Index == input.PreviousOutPoint.Index {
				inval += utxo.Value
				break
			}
		}
	}
	if inval < outval {
		return 0, errors.New("error building transaction: inputs less than outputs")
	}
	return uint64(inval - outval), err
}

func (w *BtcElectrumWallet) buildTx(amount int64, addr btcutil.Address, feeLevel wallet.FeeLevel, optionalOutput *wire.TxOut) (*wire.MsgTx, error) {
	// Check for dust
	script, _ := txscript.PayToAddrScript(addr)
	if w.IsDust(amount) {
		return nil, wallet.ErrDustAmount
	}

	var additionalPrevScripts map[wire.OutPoint][]byte
	var additionalKeysByAddress map[string]*btcutil.WIF

	// Create input source
	coinMap := w.gatherCoins()
	coins := make([]coinset.Coin, 0, len(coinMap))
	for k := range coinMap {
		coins = append(coins, k)
	}

	inputSource := func(target btcutil.Amount) (total btcutil.Amount, inputs []*wire.TxIn, inputValues []btcutil.Amount, scripts [][]byte, err error) {
		coinSelector := coinset.MaxValueAgeCoinSelector{MaxInputs: 10000, MinChangeAmount: btcutil.Amount(0)}
		coins, err := coinSelector.CoinSelect(target, coins)
		if err != nil {
			return total, inputs, []btcutil.Amount{}, scripts, wallet.ErrInsufficientFunds
		}
		additionalPrevScripts = make(map[wire.OutPoint][]byte)
		additionalKeysByAddress = make(map[string]*btcutil.WIF)
		for _, c := range coins.Coins() {
			total += c.Value()
			outpoint := wire.NewOutPoint(c.Hash(), c.Index())
			in := wire.NewTxIn(outpoint, []byte{}, [][]byte{})
			in.Sequence = 0 // Opt-in RBF so we can bump fees
			inputs = append(inputs, in)
			additionalPrevScripts[*outpoint] = c.PkScript()
			key := coinMap[c]
			addr, err := key.Address(w.params)
			if err != nil {
				continue
			}
			privKey, err := key.ECPrivKey()
			if err != nil {
				continue
			}
			wif, _ := btcutil.NewWIF(privKey, w.params, true)
			additionalKeysByAddress[addr.EncodeAddress()] = wif
		}
		return total, inputs, []btcutil.Amount{}, scripts, nil
	}

	// Get the fee per kilobyte
	feePerKB := int64(w.GetFeePerByte(feeLevel)) * 1000

	// outputs
	out := wire.NewTxOut(amount, script)

	// Create change source
	changeSource := func() ([]byte, error) {
		address, err := w.GetUnusedAddress(wallet.INTERNAL)
		if err != nil {
			return []byte{}, err
		}
		script, err := txscript.PayToAddrScript(address)
		if err != nil {
			return []byte{}, err
		}
		return script, nil
	}
	var scriptSize int = 0
	changeOutputsSource := txauthor.ChangeSource{
		NewScript:  changeSource,
		ScriptSize: scriptSize,
	}

	outputs := []*wire.TxOut{out}
	if optionalOutput != nil {
		outputs = append(outputs, optionalOutput)
	}
	authoredTx, err := NewUnsignedTransaction(outputs, btcutil.Amount(feePerKB), inputSource, changeOutputsSource)
	if err != nil {
		return nil, err
	}

	// BIP 69 sorting
	txsort.InPlaceSort(authoredTx.Tx)

	// Sign tx
	getKey := txscript.KeyClosure(func(addr btcutil.Address) (*btcec.PrivateKey, bool, error) {
		addrStr := addr.EncodeAddress()
		wif := additionalKeysByAddress[addrStr]
		return wif.PrivKey, wif.CompressPubKey, nil
	})
	getScript := txscript.ScriptClosure(func(
		addr btcutil.Address) ([]byte, error) {
		return []byte{}, nil
	})
	for i, txIn := range authoredTx.Tx.TxIn {
		prevOutScript := additionalPrevScripts[txIn.PreviousOutPoint]
		script, err := txscript.SignTxOutput(w.params,
			authoredTx.Tx, i, prevOutScript, txscript.SigHashAll, getKey,
			getScript, txIn.SignatureScript)
		if err != nil {
			return nil, errors.New("failed to sign transaction")
		}
		txIn.SignatureScript = script
	}
	return authoredTx.Tx, nil
}

func (w *BtcElectrumWallet) buildSpendAllTx(addr btcutil.Address, feeLevel wallet.FeeLevel) (*wire.MsgTx, error) {
	tx := wire.NewMsgTx(1)

	coinMap := w.gatherCoins()
	inVals := make(map[wire.OutPoint]int64)
	totalIn := int64(0)
	additionalPrevScripts := make(map[wire.OutPoint][]byte)
	additionalKeysByAddress := make(map[string]*btcutil.WIF)

	for coin, key := range coinMap {
		outpoint := wire.NewOutPoint(coin.Hash(), coin.Index())
		in := wire.NewTxIn(outpoint, nil, nil)
		additionalPrevScripts[*outpoint] = coin.PkScript()
		tx.TxIn = append(tx.TxIn, in)
		val := int64(coin.Value().ToUnit(btcutil.AmountSatoshi))
		totalIn += val
		inVals[*outpoint] = val

		addr, err := key.Address(w.params)
		if err != nil {
			continue
		}
		privKey, err := key.ECPrivKey()
		if err != nil {
			continue
		}
		wif, _ := btcutil.NewWIF(privKey, w.params, true)
		additionalKeysByAddress[addr.EncodeAddress()] = wif
	}

	// outputs
	script, err := txscript.PayToAddrScript(addr)
	if err != nil {
		return nil, err
	}

	// Get the fee
	feePerByte := int64(w.GetFeePerByte(feeLevel))
	estimatedSize := EstimateSerializeSize(1, []*wire.TxOut{wire.NewTxOut(0, script)}, false, P2PKH)
	fee := int64(estimatedSize) * feePerByte

	// Check for dust output
	if w.IsDust(totalIn - fee) {
		return nil, wallet.ErrDustAmount
	}

	// Build the output
	out := wire.NewTxOut(totalIn-fee, script)
	tx.TxOut = append(tx.TxOut, out)

	// BIP 69 sorting
	txsort.InPlaceSort(tx)

	// Sign
	getKey := txscript.KeyClosure(func(addr btcutil.Address) (*btcec.PrivateKey, bool, error) {
		addrStr := addr.EncodeAddress()
		wif, ok := additionalKeysByAddress[addrStr]
		if !ok {
			return nil, false, errors.New("key not found")
		}
		return wif.PrivKey, wif.CompressPubKey, nil
	})
	getScript := txscript.ScriptClosure(func(
		addr btcutil.Address) ([]byte, error) {
		return []byte{}, nil
	})
	for i, txIn := range tx.TxIn {
		prevOutScript := additionalPrevScripts[txIn.PreviousOutPoint]
		script, err := txscript.SignTxOutput(w.params,
			tx, i, prevOutScript, txscript.SigHashAll, getKey,
			getScript, txIn.SignatureScript)
		if err != nil {
			return nil, errors.New("failed to sign transaction")
		}
		txIn.SignatureScript = script
	}
	return tx, nil
}

func isDust(amount int64) bool {
	return btcutil.Amount(amount) < txrules.DefaultRelayFeePerKb
}

func NewUnsignedTransaction(outputs []*wire.TxOut, feePerKb btcutil.Amount, fetchInputs txauthor.InputSource, changeSource txauthor.ChangeSource) (*txauthor.AuthoredTx, error) {
	var targetAmount btcutil.Amount
	for _, txOut := range outputs {
		targetAmount += btcutil.Amount(txOut.Value)
	}

	estimatedSize := EstimateSerializeSize(1, outputs, true, P2PKH)
	targetFee := txrules.FeeForSerializeSize(feePerKb, estimatedSize)

	for {
		inputAmount, inputs, _, scripts, err := fetchInputs(targetAmount + targetFee)
		if err != nil {
			return nil, err
		}
		if inputAmount < targetAmount+targetFee {
			return nil, errors.New("insufficient funds available to construct transaction")
		}

		maxSignedSize := EstimateSerializeSize(len(inputs), outputs, true, P2PKH)
		maxRequiredFee := txrules.FeeForSerializeSize(feePerKb, maxSignedSize)
		remainingAmount := inputAmount - targetAmount
		if remainingAmount < maxRequiredFee {
			targetFee = maxRequiredFee
			continue
		}

		unsignedTransaction := &wire.MsgTx{
			Version:  wire.TxVersion,
			TxIn:     inputs,
			TxOut:    outputs,
			LockTime: 0,
		}
		changeIndex := -1
		changeAmount := inputAmount - targetAmount - maxRequiredFee
		if changeAmount != 0 && !isDust(int64(changeAmount)) {
			changeScript, err := changeSource.NewScript()
			if err != nil {
				return nil, err
			}
			if len(changeScript) > P2PKHPkScriptSize {
				return nil, errors.New("fee estimation requires change " +
					"scripts no larger than P2PKH output scripts")
			}
			change := wire.NewTxOut(int64(changeAmount), changeScript)
			l := len(outputs)
			unsignedTransaction.TxOut = append(outputs[:l:l], change)
			changeIndex = l
		}

		return &txauthor.AuthoredTx{
			Tx:          unsignedTransaction,
			PrevScripts: scripts,
			TotalInput:  inputAmount,
			ChangeIndex: changeIndex,
		}, nil
	}
}

func (w *BtcElectrumWallet) GetFeePerByte(feeLevel wallet.FeeLevel) int64 {
	return w.feeProvider.GetFeePerByte(feeLevel)
}

func LockTimeFromRedeemScript(redeemScript []byte) (uint32, error) {
	if len(redeemScript) < 113 {
		return 0, errors.New("redeem script invalid length")
	}
	if redeemScript[106] != 103 {
		return 0, errors.New("rnvalid redeem script")
	}
	if redeemScript[107] == 0 {
		return 0, nil
	}
	if 81 <= redeemScript[107] && redeemScript[107] <= 96 {
		return uint32((redeemScript[107] - 81) + 1), nil
	}
	var v []byte
	op := redeemScript[107]
	if 1 <= op && op <= 75 {
		for i := 0; i < int(op); i++ {
			v = append(v, []byte{redeemScript[108+i]}...)
		}
	} else {
		return 0, errors.New("too many bytes pushed for sequence")
	}
	var result int64
	for i, val := range v {
		result |= int64(val) << uint8(8*i)
	}

	return uint32(result), nil
}
