package wltbtc

import (
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

// Multisig functionality here

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

func (w *BtcElectrumWallet) CreateMultisigSignature(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, key *hdkeychain.ExtendedKey, redeemScript []byte, feePerByte int64) ([]wallet.Signature, error) {

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

func (w *BtcElectrumWallet) Multisign(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, sigs1 []wallet.Signature, sigs2 []wallet.Signature, redeemScript []byte, feePerByte int64, broadcast bool) ([]byte, error) {

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
