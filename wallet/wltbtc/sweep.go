package wltbtc

import (
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/wire"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

// Build a transaction that sweeps all coins from a private key.

// https://99bitcoins.com/bitcoin-wallet/paper/private-key-sweep-import

func (w *BtcElectrumWallet) SweepCoins(coins []wallet.Utxo, address btcutil.Address, key *hdkeychain.ExtendedKey, redeemScript []byte, feeLevel wallet.FeeLevel) (*wire.MsgTx, error) {

	return &wire.MsgTx{}, nil
}

// If it is a p2sh multisig, the redeemScript must be included
// Tx is returned ready to broadcast. Only the client can broadcast (via Node)
// func (w *BtcElectrumWallet) SweepAddress(ins []wallet.TransactionInput, address btcutil.Address, key *hdkeychain.ExtendedKey, redeemScript []byte, feeLevel wallet.FeeLevel) (*wire.MsgTx, error) {

// 	return nil, wallet.ErrWalletFnNotImplemented // FIXME: SweepAddress prevOutputFInder

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

// ready to broadcast -- only client can broadcast

// return &tx, nil
// }
