package wltbtc

import (
	"bytes"
	"errors"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

// Sign an unsigned transaction with the wallet
func (w *BtcElectrumWallet) SignTx(pw string, info *wallet.SigningInfo) ([]byte, error) {
	if ok := w.storageManager.IsValidPw(pw); !ok {
		return nil, errors.New("invalid password")
	}
	tx := info.UnsignedTx
	prevOutFetcher := new(txscript.CannedPrevOutputFetcher)
	sigHashes := txscript.NewTxSigHashes(tx, prevOutFetcher)
	address, err := w.GetUnusedAddress(wallet.RECEIVING)
	if err != nil {
		return nil, err
	}
	key, err := w.txstore.keyManager.GetKeyForScript(address.ScriptAddress())
	if err != nil {
		return nil, err
	}
	defer key.Zero()
	privKey, err := key.ECPrivKey()
	if err != nil {
		return nil, err
	}
	for i, prevOut := range info.PrevOuts {
		sig, pubKey, err := createWitnessSig(tx, i, prevOut.Value, prevOut.RedeemScript, sigHashes, privKey)
		if err != nil {
			return nil, err
		}
		tx.TxIn[i].Witness = make([][]byte, 0, 2)
		tx.TxIn[i].Witness = append(tx.TxIn[i].Witness, sig, pubKey)
		if info.VerifyTx {
			e, err := txscript.NewEngine(
				// pubkey script
				prevOut.RedeemScript,
				// refund transaction
				tx,
				// transaction input index
				i,
				txscript.StandardVerifyFlags,
				txscript.NewSigCache(10),
				txscript.NewTxSigHashes(tx, prevOutFetcher),
				prevOut.Value,
				prevOutFetcher)
			if err != nil {
				panic(err)
			}
			err = e.Execute()
			if err != nil {
				panic(err)
			}
		}
	}
	b := make([]byte, 0)
	txOut := bytes.NewBuffer(b)
	err = tx.Serialize(txOut)
	if err != nil {
		return nil, err
	}
	return txOut.Bytes(), nil
}

// createWitnessSig creates and returns the serialized raw signature and compressed
// pubkey for a transaction input signature.
//
// returns [sig][pubkey]
func createWitnessSig(
	tx *wire.MsgTx,
	idx int,
	prevOutValue int64,
	prevOutPkScript []byte,
	sigHashes *txscript.TxSigHashes,
	privKey *secp256k1.PrivateKey) ([]byte, []byte, error) {

	sig, err := txscript.RawTxInWitnessSignature(tx, sigHashes, idx, prevOutValue,
		prevOutPkScript, txscript.SigHashAll, privKey)
	if err != nil {
		return nil, nil, err
	}

	return sig, privKey.PubKey().SerializeCompressed(), nil
}
