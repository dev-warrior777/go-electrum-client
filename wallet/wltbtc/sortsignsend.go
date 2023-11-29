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

func (w *BtcElectrumWallet) Spend(amount int64, addr btcutil.Address, feeLevel wallet.FeeLevel, referenceID string, spendAll bool) (*chainhash.Hash, error) {
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

	if err := w.Broadcast(tx); err != nil {
		return nil, err
	}
	ch := tx.TxHash()
	return &ch, nil
}

// var BumpFeeAlreadyConfirmedError = errors.New("transaction is confirmed, cannot bump fee")
// var BumpFeeTransactionDeadError = errors.New("cannot bump fee of dead transaction")
// var BumpFeeNotFoundError = errors.New("transaction either doesn't exist or has already been spent")

func (w *BtcElectrumWallet) BumpFee(txid chainhash.Hash) (*chainhash.Hash, error) {

	return nil, wallet.ErrWalletFnNotImplemented // FIXME: SweepAddress prevOutputFInder

	// txn, err := w.txstore.Txns().Get(txid)
	// if err != nil {
	// 	return nil, err
	// }
	// if txn.Height > 0 {
	// 	return nil, BumpFeeAlreadyConfirmedError
	// }
	// if txn.Height < 0 {
	// 	return nil, BumpFeeTransactionDeadError
	// }

	// // As a policy this wallet will never do RBF.

	// // Check utxos for CPFP
	// utxos, _ := w.txstore.Utxos().GetAll()
	// for _, u := range utxos {
	// 	if u.Op.Hash.IsEqual(&txid) && u.AtHeight == 0 {
	// 		addr, err := w.ScriptToAddress(u.ScriptPubkey)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		key, err := w.keyManager.GetKeyForScript(addr.ScriptAddress())
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		in := wallet.TransactionInput{
	// 			LinkedAddress: addr,
	// 			OutpointIndex: u.Op.Index,
	// 			OutpointHash:  u.Op.Hash.CloneBytes(),
	// 			Value:         u.Value,
	// 		}

	// 		newAddress, err := w.GetUnusedAddress(wallet.INTERNAL)
	// 		if err != nil {
	// 			return nil, err
	// 		}

	// 		transactionID, err := w.SweepAddress([]wallet.TransactionInput{in}, newAddress, key, nil, wallet.FEE_BUMP)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		return transactionID, nil
	// 	}
	// }
	// return nil, BumpFeeNotFoundError
}

func (w *BtcElectrumWallet) EstimateFee(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, feePerByte uint64) uint64 {
	tx := new(wire.MsgTx)
	for _, out := range outs {
		scriptPubKey, _ := txscript.PayToAddrScript(out.Address)
		output := wire.NewTxOut(out.Value, scriptPubKey)
		tx.TxOut = append(tx.TxOut, output)
	}
	estimatedSize := EstimateSerializeSize(len(ins), tx.TxOut, false, P2PKH)
	fee := estimatedSize * int(feePerByte)
	return uint64(fee)
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

func (w *BtcElectrumWallet) GenerateMultisigScript(keys []hdkeychain.ExtendedKey, threshold int, timeout time.Duration, timeoutKey *hdkeychain.ExtendedKey) (addr btcutil.Address, redeemScript []byte, err error) {

	return nil, nil, wallet.ErrWalletFnNotImplemented

	// if uint32(timeout.Hours()) > 0 && timeoutKey == nil {
	// 	return nil, nil, errors.New("timeout key must be non nil when using an escrow timeout")
	// }

	// if len(keys) < threshold {
	// 	return nil, nil, fmt.Errorf("unable to generate multisig script with "+
	// 		"%d required signatures when there are only %d public "+
	// 		"keys available", threshold, len(keys))
	// }

	// var ecKeys []*btcec.PublicKey
	// for _, key := range keys {
	// 	ecKey, err := key.ECPubKey()
	// 	if err != nil {
	// 		return nil, nil, err
	// 	}
	// 	ecKeys = append(ecKeys, ecKey)
	// }

	// builder := txscript.NewScriptBuilder()
	// if uint32(timeout.Hours()) == 0 {

	// 	builder.AddInt64(int64(threshold))
	// 	for _, key := range ecKeys {
	// 		builder.AddData(key.SerializeCompressed())
	// 	}
	// 	builder.AddInt64(int64(len(ecKeys)))
	// 	builder.AddOp(txscript.OP_CHECKMULTISIG)

	// } else {
	// 	ecKey, err := timeoutKey.ECPubKey()
	// 	if err != nil {
	// 		return nil, nil, err
	// 	}
	// 	sequenceLock := blockchain.LockTimeToSequence(false, uint32(timeout.Hours()*6))
	// 	builder.AddOp(txscript.OP_IF)
	// 	builder.AddInt64(int64(threshold))
	// 	for _, key := range ecKeys {
	// 		builder.AddData(key.SerializeCompressed())
	// 	}
	// 	builder.AddInt64(int64(len(ecKeys)))
	// 	builder.AddOp(txscript.OP_CHECKMULTISIG)
	// 	builder.AddOp(txscript.OP_ELSE).
	// 		AddInt64(int64(sequenceLock)).
	// 		AddOp(txscript.OP_CHECKSEQUENCEVERIFY).
	// 		AddOp(txscript.OP_DROP).
	// 		AddData(ecKey.SerializeCompressed()).
	// 		AddOp(txscript.OP_CHECKSIG).
	// 		AddOp(txscript.OP_ENDIF)
	// }
	// redeemScript, err = builder.Script()
	// if err != nil {
	// 	return nil, nil, err
	// }

	// witnessProgram := sha256.Sum256(redeemScript)

	// addr, err = btcutil.NewAddressWitnessScriptHash(witnessProgram[:], w.params)
	// if err != nil {
	// 	return nil, nil, err
	// }
	// return addr, redeemScript, nil
}

func (w *BtcElectrumWallet) CreateMultisigSignature(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, key *hdkeychain.ExtendedKey, redeemScript []byte, feePerByte uint64) ([]wallet.Signature, error) {

	return nil, wallet.ErrWalletFnNotImplemented

	// var sigs []wallet.Signature
	// tx := wire.NewMsgTx(1)
	// for _, in := range ins {
	// 	ch, err := chainhash.NewHashFromStr(hex.EncodeToString(in.OutpointHash))
	// 	if err != nil {
	// 		return sigs, err
	// 	}
	// 	outpoint := wire.NewOutPoint(ch, in.OutpointIndex)
	// 	input := wire.NewTxIn(outpoint, []byte{}, [][]byte{})
	// 	tx.TxIn = append(tx.TxIn, input)
	// }
	// for _, out := range outs {
	// 	scriptPubKey, err := txscript.PayToAddrScript(out.Address)
	// 	if err != nil {
	// 		return sigs, err
	// 	}
	// 	output := wire.NewTxOut(out.Value, scriptPubKey)
	// 	tx.TxOut = append(tx.TxOut, output)
	// }

	// // Subtract fee
	// txType := P2SH_2of3_Multisig
	// _, err := LockTimeFromRedeemScript(redeemScript)
	// if err == nil {
	// 	txType = P2SH_Multisig_Timelock_2Sigs
	// }
	// estimatedSize := EstimateSerializeSize(len(ins), tx.TxOut, false, txType)
	// fee := estimatedSize * int(feePerByte)
	// if len(tx.TxOut) > 0 {
	// 	feePerOutput := fee / len(tx.TxOut)
	// 	for _, output := range tx.TxOut {
	// 		output.Value -= int64(feePerOutput)
	// 	}
	// }

	// // BIP 69 sorting
	// txsort.InPlaceSort(tx)

	// signingKey, err := key.ECPrivKey()
	// if err != nil {
	// 	return sigs, err
	// }

	// hashes := txscript.NewTxSigHashes(tx, nil)
	// for i := range tx.TxIn {
	// 	sig, err := txscript.RawTxInWitnessSignature(tx, hashes, i, ins[i].Value, redeemScript, txscript.SigHashAll, signingKey)
	// 	if err != nil {
	// 		continue
	// 	}
	// 	bs := wallet.Signature{InputIndex: uint32(i), Signature: sig}
	// 	sigs = append(sigs, bs)
	// }
	// return sigs, nil
}

func (w *BtcElectrumWallet) Multisign(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, sigs1 []wallet.Signature, sigs2 []wallet.Signature, redeemScript []byte, feePerByte uint64, broadcast bool) ([]byte, error) {

	return nil, wallet.ErrWalletFnNotImplemented

	// tx := wire.NewMsgTx(1)
	// for _, in := range ins {
	// 	ch, err := chainhash.NewHashFromStr(hex.EncodeToString(in.OutpointHash))
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	outpoint := wire.NewOutPoint(ch, in.OutpointIndex)
	// 	input := wire.NewTxIn(outpoint, []byte{}, [][]byte{})
	// 	tx.TxIn = append(tx.TxIn, input)
	// }
	// for _, out := range outs {
	// 	scriptPubKey, err := txscript.PayToAddrScript(out.Address)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	output := wire.NewTxOut(out.Value, scriptPubKey)
	// 	tx.TxOut = append(tx.TxOut, output)
	// }

	// // Subtract fee
	// txType := P2SH_2of3_Multisig
	// _, err := LockTimeFromRedeemScript(redeemScript)
	// if err == nil {
	// 	txType = P2SH_Multisig_Timelock_2Sigs
	// }
	// estimatedSize := EstimateSerializeSize(len(ins), tx.TxOut, false, txType)
	// fee := estimatedSize * int(feePerByte)
	// if len(tx.TxOut) > 0 {
	// 	feePerOutput := fee / len(tx.TxOut)
	// 	for _, output := range tx.TxOut {
	// 		output.Value -= int64(feePerOutput)
	// 	}
	// }

	// // BIP 69 sorting
	// txsort.InPlaceSort(tx)

	// // Check if time locked
	// var timeLocked bool
	// if redeemScript[0] == txscript.OP_IF {
	// 	timeLocked = true
	// }

	// for i, input := range tx.TxIn {
	// 	var sig1 []byte
	// 	var sig2 []byte
	// 	for _, sig := range sigs1 {
	// 		if int(sig.InputIndex) == i {
	// 			sig1 = sig.Signature
	// 			break
	// 		}
	// 	}
	// 	for _, sig := range sigs2 {
	// 		if int(sig.InputIndex) == i {
	// 			sig2 = sig.Signature
	// 			break
	// 		}
	// 	}

	// 	witness := wire.TxWitness{[]byte{}, sig1, sig2}

	// 	if timeLocked {
	// 		witness = append(witness, []byte{0x01})
	// 	}
	// 	witness = append(witness, redeemScript)
	// 	input.Witness = witness
	// }
	// // broadcast
	// if broadcast {
	// 	w.Broadcast(tx)
	// }
	// var buf bytes.Buffer
	// tx.BtcEncode(&buf, wire.ProtocolVersion, wire.WitnessEncoding)
	// return buf.Bytes(), nil
}

// Build a transaction that sweeps all coins from an address.
// If it is a p2sh multisig, the redeemScript must be included
func (w *BtcElectrumWallet) SweepAddress(ins []wallet.TransactionInput, address btcutil.Address, key *hdkeychain.ExtendedKey, redeemScript []byte, feeLevel wallet.FeeLevel) (*chainhash.Hash, error) {

	return nil, wallet.ErrWalletFnNotImplemented // FIXME: SweepAddress prevOutputFInder

	// if address == nil {
	// 	return nil, errors.New("empty address")
	// }

	// script, err := txscript.PayToAddrScript(address)
	// if err != nil {
	// 	return nil, err
	// }

	// var val int64
	// var inputs []*wire.TxIn
	// additionalPrevScripts := make(map[wire.OutPoint][]byte)
	// for _, in := range ins {
	// 	val += in.Value
	// 	ch, err := chainhash.NewHashFromStr(hex.EncodeToString(in.OutpointHash))
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	script, err := txscript.PayToAddrScript(in.LinkedAddress)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	outpoint := wire.NewOutPoint(ch, in.OutpointIndex)
	// 	input := wire.NewTxIn(outpoint, []byte{}, [][]byte{})
	// 	inputs = append(inputs, input)
	// 	additionalPrevScripts[*outpoint] = script
	// }
	// out := wire.NewTxOut(val, script)

	// txType := P2PKH
	// if redeemScript != nil {
	// 	txType = P2SH_1of2_Multisig
	// 	_, err := LockTimeFromRedeemScript(redeemScript)
	// 	if err == nil {
	// 		txType = P2SH_Multisig_Timelock_1Sig
	// 	}
	// }
	// estimatedSize := EstimateSerializeSize(len(ins), []*wire.TxOut{out}, false, txType)

	// // Calculate the fee
	// feePerByte := int(w.GetFeePerByte(feeLevel))
	// fee := estimatedSize * feePerByte

	// outVal := val - int64(fee)
	// if outVal < 0 {
	// 	outVal = 0
	// }
	// out.Value = outVal

	// tx := &wire.MsgTx{
	// 	Version:  wire.TxVersion,
	// 	TxIn:     inputs,
	// 	TxOut:    []*wire.TxOut{out},
	// 	LockTime: 0,
	// }

	// // BIP 69 sorting
	// txsort.InPlaceSort(tx)

	// // Sign tx
	// privKey, err := key.ECPrivKey()
	// if err != nil {
	// 	return nil, err
	// }
	// pk := privKey.PubKey().SerializeCompressed()
	// addressPub, err := btcutil.NewAddressPubKey(pk, w.params)

	// getKey := txscript.KeyClosure(func(addr btcutil.Address) (*btcec.PrivateKey, bool, error) {
	// 	if addressPub.EncodeAddress() == addr.EncodeAddress() {
	// 		wif, err := btcutil.NewWIF(privKey, w.params, true)
	// 		if err != nil {
	// 			return nil, false, err
	// 		}
	// 		return wif.PrivKey, wif.CompressPubKey, nil
	// 	}
	// 	return nil, false, errors.New("not found")
	// })
	// getScript := txscript.ScriptClosure(func(addr btcutil.Address) ([]byte, error) {
	// 	if redeemScript == nil {
	// 		return []byte{}, nil
	// 	}
	// 	return redeemScript, nil
	// })

	// // Check if time locked
	// var timeLocked bool
	// if redeemScript != nil {
	// 	rs := redeemScript
	// 	if rs[0] == txscript.OP_IF {
	// 		timeLocked = true
	// 		tx.Version = 2
	// 		for _, txIn := range tx.TxIn {
	// 			locktime, err := LockTimeFromRedeemScript(redeemScript)
	// 			if err != nil {
	// 				return nil, err
	// 			}
	// 			txIn.Sequence = locktime
	// 		}
	// 	}
	// }

	// //FIXME:
	// fetcher := txscript.NewCannedPrevOutputFetcher(
	// 	utxOut.PkScript, utxOut.Value,
	// )
	// hashes := txscript.NewTxSigHashes(tx, fetcher)
	// for i, txIn := range tx.TxIn {
	// 	if redeemScript == nil {
	// 		prevOutScript := additionalPrevScripts[txIn.PreviousOutPoint]
	// 		script, err := txscript.SignTxOutput(w.params,
	// 			tx, i, prevOutScript, txscript.SigHashAll, getKey,
	// 			getScript, txIn.SignatureScript)
	// 		if err != nil {
	// 			return nil, errors.New("failed to sign transaction")
	// 		}
	// 		txIn.SignatureScript = script
	// 	} else {
	// 		sig, err := txscript.RawTxInWitnessSignature(tx, hashes, i, ins[i].Value, redeemScript, txscript.SigHashAll, privKey)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		var witness wire.TxWitness
	// 		if timeLocked {
	// 			witness = wire.TxWitness{sig, []byte{}}
	// 		} else {
	// 			witness = wire.TxWitness{[]byte{}, sig}
	// 		}
	// 		witness = append(witness, redeemScript)
	// 		txIn.Witness = witness
	// 	}
	// }

	// // broadcast
	// w.Broadcast(tx)
	// txid := tx.TxHash()
	// return &txid, nil
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

func (w *BtcElectrumWallet) GetFeePerByte(feeLevel wallet.FeeLevel) uint64 {
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
