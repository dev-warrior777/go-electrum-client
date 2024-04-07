package btc

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

// Import UTXO's for a known privkey from another wallet from electrumX. Partially
// implemented (P2WPKH,P2PKH) as it is not the most important tool for this wallet.

func (ec *BtcElectrumClient) getWitnessScriptHashRedeemUtxos( /*ctx*/ context.Context /*keyPair*/, *btcutil.WIF) ([]wallet.InputInfo, error) {
	utxoList := make([]wallet.InputInfo, 0)
	fmt.Println("looking for P2WSH redemptions - Not implemented")
	return utxoList, nil
}

func (ec *BtcElectrumClient) getScriptHashRedeemUtxos( /*ctx*/ context.Context /*keyPair*/, *btcutil.WIF) ([]wallet.InputInfo, error) {
	utxoList := make([]wallet.InputInfo, 0)
	fmt.Println("looking for P2SH Redemptions  - Not implemented")
	return utxoList, nil
}

func (ec *BtcElectrumClient) getPubKeyUtxos( /*ctx*/ context.Context /*keyPair*/, *btcutil.WIF) ([]wallet.InputInfo, error) {
	utxoList := make([]wallet.InputInfo, 0)
	fmt.Println("looking for P2PK - Not implemented")
	return utxoList, nil
}

func (ec *BtcElectrumClient) getPubKeyHashUtxos(ctx context.Context, keyPair *btcutil.WIF) ([]wallet.InputInfo, error) {
	inputList := make([]wallet.InputInfo, 0, 1)
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
		return inputList, err
	}
	// ask electrumX
	listUnspent, err := node.GetListUnspent(ctx, scripthash)
	if err != nil {
		return inputList, err
	}
	for _, unspent := range listUnspent {
		op, err := wallet.NewOutPointFromString(
			fmt.Sprintf("%s:%d", unspent.TxHash, unspent.TxPos))
		if err != nil {
			return inputList, err
		}
		input := wallet.InputInfo{
			Outpoint:      op,
			Height:        unspent.Height,
			Value:         unspent.Value,
			LinkedAddress: addressPubKeyHash,
			PkScript:      []byte{},
			KeyPair:       keyPair,
		}
		inputList = append(inputList, input)
	}
	return inputList, nil
}

func (ec *BtcElectrumClient) getWitnessPubKeyHashUtxos(ctx context.Context, keyPair *btcutil.WIF) ([]wallet.InputInfo, error) {
	inputList := make([]wallet.InputInfo, 0, 1)
	pubKey := keyPair.SerializePubKey()

	node := ec.GetNode()
	if node == nil {
		return inputList, ErrNoNode
	}
	// make address p2wpkh
	pkHash := btcutil.Hash160(pubKey)
	addressWitnessPubKeyHash, err := btcutil.NewAddressWitnessPubKeyHash(pkHash, ec.GetConfig().Params)
	if err != nil {
		return inputList, err
	}
	// make scripthash
	scripthash, err := addressToElectrumScripthash(addressWitnessPubKeyHash)
	if err != nil {
		return inputList, err
	}
	// ask electrum
	listUnspent, err := node.GetListUnspent(ctx, scripthash)
	if err != nil {
		return inputList, err
	}
	for _, unspent := range listUnspent {
		op, err := wallet.NewOutPointFromString(
			fmt.Sprintf("%s:%d", unspent.TxHash, unspent.TxPos))
		if err != nil {
			return inputList, err
		}
		input := wallet.InputInfo{
			Outpoint:      op,
			Height:        unspent.Height,
			Value:         unspent.Value,
			LinkedAddress: addressWitnessPubKeyHash,
			PkScript:      []byte{},
			KeyPair:       keyPair,
		}
		inputList = append(inputList, input)
	}
	return inputList, nil
}

func (ec *BtcElectrumClient) getUtxos(ctx context.Context, keyPair *btcutil.WIF) ([]wallet.InputInfo, error) {
	inputList := make([]wallet.InputInfo, 0, 1)

	// P2WSH Redeem - not yet implemented
	p2wshInputList, err := ec.getWitnessScriptHashRedeemUtxos(ctx, keyPair)
	if err != nil {
		return inputList, err
	}
	fmt.Printf("found %d P2WSH Redemption utxos\n", len(p2wshInputList))
	if len(p2wshInputList) > 0 {
		inputList = append(inputList, p2wshInputList...)
	}

	// P2SH Redeem - not yet implemented
	p2shInputList, err := ec.getScriptHashRedeemUtxos(ctx, keyPair)
	if err != nil {
		return inputList, err
	}
	fmt.Printf("found %d P2SH Redemption utxos\n", len(p2shInputList))
	if len(p2shInputList) > 0 {
		inputList = append(inputList, p2shInputList...)
	}

	// P2PK - including satoshi's coins maybe .. not yet implemented
	p2pkInputList, err := ec.getPubKeyUtxos(ctx, keyPair)
	if err != nil {
		return inputList, err
	}
	fmt.Printf("found %d P2PK utxos\n", len(p2pkInputList))
	if len(p2pkInputList) > 0 {
		inputList = append(inputList, p2pkInputList...)
	}

	// P2PKH
	p2pkhInputList, err := ec.getPubKeyHashUtxos(ctx, keyPair)
	if err != nil {
		return inputList, err
	}
	fmt.Printf("found %d P2PKH utxos\n", len(p2pkhInputList))
	if len(p2pkhInputList) > 0 {
		inputList = append(inputList, p2pkhInputList...)
	}

	// P2WPKH
	p2wpkhInputList, err := ec.getWitnessPubKeyHashUtxos(ctx, keyPair)
	if err != nil {
		return inputList, err
	}
	fmt.Printf("found %d P2WPKH utxos\n", len(p2wpkhInputList))
	if len(p2wpkhInputList) > 0 {
		inputList = append(inputList, p2wpkhInputList...)
	}

	return inputList, nil
}

// ImportAndSweep imports privkeys from other wallets and builds a transaction that
// sweeps their funds into our wallet.
func (ec *BtcElectrumClient) ImportAndSweep(ctx context.Context, importedKeyPairs []string) error {
	w := ec.GetWallet()
	if w == nil {
		return ErrNoWallet
	}
	if len(importedKeyPairs) <= 0 {
		return errors.New("no keys")
	}
	var inputs []wallet.InputInfo = make([]wallet.InputInfo, 0)
	for _, k := range importedKeyPairs {
		wif, err := btcutil.DecodeWIF(k)
		if err != nil {
			fmt.Printf("warning cannot decode WIF from string: %s\n", k)
			continue
		}

		inputsForKey, err := ec.getUtxos(ctx, wif)
		if err != nil {
			fmt.Printf("warning cannot get utxos for pubkey: %s\n",
				hex.EncodeToString(wif.SerializePubKey()))
			continue
		}
		if len(inputsForKey) <= 0 {
			continue
		}
		inputs = append(inputs, inputsForKey...)
	}
	if len(inputs) <= 0 {
		return errors.New("no inputs found")
	}
	// wallet sweep []tx
	txs, err := w.SweepCoins(inputs, wallet.NORMAL, 50)
	if err != nil {
		return err
	}
	for _, tx := range txs {
		var sweepBuf bytes.Buffer
		sweepBuf.Grow(tx.SerializeSize())
		tx.Serialize(&sweepBuf)
		fmt.Printf("Sweep Tx:      %x\n\n", sweepBuf.Bytes())
		fmt.Printf("Sweep TxHash: (%v):\n", tx.TxHash())
	}

	return nil
}
