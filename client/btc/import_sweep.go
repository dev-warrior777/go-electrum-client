package btc

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/wire"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

func (ec *BtcElectrumClient) getPubKeyUtxos(keyPair *btcutil.WIF) ([]wallet.TransactionInput, error) {
	utxoList := make([]wallet.TransactionInput, 0)
	fmt.Println(keyPair)
	return utxoList, nil
}

func (ec *BtcElectrumClient) getPubKeyHashUtxos(keyPair *btcutil.WIF) ([]wallet.TransactionInput, error) {
	inputList := make([]wallet.TransactionInput, 0, 1)
	pubKey := keyPair.SerializePubKey()

	node := ec.GetNode()
	if node == nil {
		return inputList, ErrNoNode
	}
	// make address p2pkh
	pkHash := btcutil.Hash160(pubKey)
	addressPubKeyHash, err := btcutil.NewAddressPubKeyHash(pkHash, ec.GetConfig().Params)
	if err != nil {
		return inputList, err
	}
	// make scripthash
	scripthash, err := addressToElectrumScripthash(addressPubKeyHash)
	if err != nil {
		fmt.Printf("cannot make script hash for address: %s\n", addressPubKeyHash.String())
		return inputList, err
	}
	fmt.Printf("address: %s scriptHash: %s\n", addressPubKeyHash.String(), scripthash)
	// ask electrum
	listUnspent, err := node.GetListUnspent(scripthash)
	if err != nil {
		return inputList, err
	}
	for _, unspent := range listUnspent {
		op, err := wire.NewOutPointFromString(
			fmt.Sprintf("%s:%d", unspent.TxHash, unspent.TxPos))
		if err != nil {
			return inputList, err
		}
		input := wallet.TransactionInput{
			Outpoint:      op,
			Height:        unspent.Height,
			Value:         unspent.Value,
			LinkedAddress: addressPubKeyHash,
			RedeemScript:  []byte{},
			KeyPair:       keyPair,
		}
		inputList = append(inputList, input)
	}
	return inputList, nil
}

func (ec *BtcElectrumClient) getWitnessPubKeyHashUtxos(keyPair *btcutil.WIF) ([]wallet.TransactionInput, error) {
	utxoList := make([]wallet.TransactionInput, 0, 1)
	fmt.Println(keyPair)
	return utxoList, nil
}

func (ec *BtcElectrumClient) getUtxos(keyPair *btcutil.WIF) ([]wallet.TransactionInput, error) {
	inputList := make([]wallet.TransactionInput, 0, 1)

	// P2PK - including satoshi's coins maybe
	p2pkInputList, err := ec.getPubKeyUtxos(keyPair)
	if err != nil {
		return inputList, err
	}
	fmt.Printf("found %d P2PK utxos\n", len(p2pkInputList))
	inputList = append(inputList, p2pkInputList...)

	// P2PKH
	p2pkhInputList, err := ec.getPubKeyHashUtxos(keyPair)
	if err != nil {
		return inputList, err
	}
	fmt.Printf("found %d P2PKH utxos\n", len(p2pkhInputList))
	inputList = append(inputList, p2pkhInputList...)

	// P2WPKH
	p2wpkhInputList, err := ec.getWitnessPubKeyHashUtxos(keyPair)
	if err != nil {
		return inputList, err
	}
	fmt.Printf("found %d P2WPKH utxos\n", len(p2wpkhInputList))
	inputList = append(inputList, p2wpkhInputList...)

	return inputList, nil
}

// ImportAndSweep imports privkeys from other wallets and builds a transaction that
// sweeps their funds into our wallet.
func (ec *BtcElectrumClient) ImportAndSweep(importedKeyPairs []string) error {
	w := ec.GetWallet()
	if w == nil {
		return ErrNoWallet
	}
	if len(importedKeyPairs) <= 0 {
		return errors.New("no keys")
	}
	for _, k := range importedKeyPairs {
		wif, err := btcutil.DecodeWIF(k)
		if err != nil {
			fmt.Printf("warning cannot decode WIF from string: %s\n", k)
			continue
		}

		inputs, err := ec.getUtxos(wif)
		if err != nil {
			fmt.Printf("warning cannot get utxos for pubkey: %s\n",
				hex.EncodeToString(wif.SerializePubKey()))
			continue
		}
		// for now just do one tx/key - later aggregate in wallet.SweepCoins.
		_, tx, err := w.SweepCoins(inputs, wallet.NORMAL)
		if err != nil {
			fmt.Printf("%v\n", err)
			continue
		}
		var sweepBuf bytes.Buffer
		sweepBuf.Grow(tx.SerializeSize())
		tx.Serialize(&sweepBuf)
		fmt.Printf("Sweep Tx:      %x\n\n", sweepBuf.Bytes())
		fmt.Printf("Sweep TxHash: (%v):\n", tx.TxHash())
	}

	return nil
}
