package btc

import (
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/wire"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

func (ec *BtcElectrumClient) getUtxosForPubKey(pubKey []byte) ([]utxoInfo, error) {
	utxoList := make([]utxoInfo, 0, 1)
	node := ec.GetNode()
	if node == nil {
		return utxoList, ErrNoNode
	}
	// make address p2pk
	addressPubKey, err := btcutil.NewAddressPubKey(pubKey, ec.GetConfig().Params)
	if err != nil {
		return utxoList, err
	}
	// make scripthash
	scripthash, err := addressToElectrumScripthash(addressPubKey)
	if err != nil {
		return utxoList, err
	}
	fmt.Printf("address: %s scriptHash: %s\n", addressPubKey.String(), scripthash)
	// ask electrum
	listUnspent, err := node.GetListUnspent(scripthash)
	if err != nil {
		return utxoList, err
	}
	for _, unspent := range listUnspent {
		op, err := wire.NewOutPointFromString(
			fmt.Sprintf("%s:%d", unspent.TxHash, unspent.TxPos))
		if err != nil {
			return utxoList, err
		}
		utxo := &wallet.Utxo{
			Op:       *op,
			AtHeight: unspent.Height,
			Value:    unspent.Value,
		}
		utxoInfo := utxoInfo{
			utxo:    utxo,
			address: addressPubKey,
		}
		utxoList = append(utxoList, utxoInfo)
	}
	return utxoList, nil
}

func (ec *BtcElectrumClient) getUtxosForPubKeyHash(pubKey []byte) ([]utxoInfo, error) {
	utxoList := make([]utxoInfo, 0, 1)
	node := ec.GetNode()
	if node == nil {
		return utxoList, ErrNoNode
	}
	// make address p2pkh
	pkHash := btcutil.Hash160(pubKey)
	addressPubKeyHash, err := btcutil.NewAddressPubKeyHash(pkHash, ec.GetConfig().Params)
	if err != nil {
		return utxoList, err
	}
	// make scripthash
	scripthash, err := addressToElectrumScripthash(addressPubKeyHash)
	if err != nil {
		fmt.Printf("cannot make script hash for address: %s\n", addressPubKeyHash.String())
		return utxoList, err
	}
	fmt.Printf("address: %s scriptHash: %s\n", addressPubKeyHash.String(), scripthash)
	// ask electrum
	listUnspent, err := node.GetListUnspent(scripthash)
	if err != nil {
		return utxoList, err
	}
	for _, unspent := range listUnspent {
		op, err := wire.NewOutPointFromString(
			fmt.Sprintf("%s:%d", unspent.TxHash, unspent.TxPos))
		if err != nil {
			return utxoList, err
		}
		utxo := &wallet.Utxo{
			Op:       *op,
			AtHeight: unspent.Height,
			Value:    unspent.Value,
		}
		utxoInfo := utxoInfo{
			utxo:    utxo,
			address: addressPubKeyHash,
		}
		utxoList = append(utxoList, utxoInfo)
	}
	return utxoList, nil
}

func (ec *BtcElectrumClient) getUtxosForWitnessPubKeyHash(pubKey []byte) ([]utxoInfo, error) {
	utxoList := make([]utxoInfo, 0, 1)
	node := ec.GetNode()
	if node == nil {
		return utxoList, ErrNoNode
	}
	// make address p2wpkh
	witnessProgram := btcutil.Hash160(pubKey)
	addressWitnessPubKeyHash, err := btcutil.NewAddressWitnessPubKeyHash(witnessProgram, ec.GetConfig().Params)
	if err != nil {
		return utxoList, err
	}
	// make scripthash
	scripthash, err := addressToElectrumScripthash(addressWitnessPubKeyHash)
	if err != nil {
		return utxoList, err
	}
	fmt.Printf("address: %s scriptHash: %s\n", addressWitnessPubKeyHash.String(), scripthash)
	// ask electrum
	listUnspent, err := node.GetListUnspent(scripthash)
	if err != nil {
		return utxoList, err
	}
	for _, unspent := range listUnspent {
		op, err := wire.NewOutPointFromString(
			fmt.Sprintf("%s:%d", unspent.TxHash, unspent.TxPos))
		if err != nil {
			return utxoList, err
		}
		utxo := &wallet.Utxo{
			Op:       *op,
			AtHeight: unspent.Height,
			Value:    unspent.Value,
		}
		utxoInfo := utxoInfo{
			utxo:    utxo,
			address: addressWitnessPubKeyHash,
		}
		utxoList = append(utxoList, utxoInfo)
	}
	return utxoList, nil
}

func (ec *BtcElectrumClient) getUtxos(pubKey []byte) ([]utxoInfo, error) {
	utxoList := make([]utxoInfo, 0, 1)

	// P2PK
	p2pkUtxoList, err := ec.getUtxosForPubKey(pubKey)
	if err != nil {
		return utxoList, err
	}
	fmt.Printf("found %d P2PK utxos\n", len(p2pkUtxoList))
	utxoList = append(utxoList, p2pkUtxoList...)

	// P2PKH
	p2pkhUtxoList, err := ec.getUtxosForPubKeyHash(pubKey)
	if err != nil {
		return utxoList, err
	}
	fmt.Printf("found %d P2PKH utxos\n", len(p2pkhUtxoList))
	utxoList = append(utxoList, p2pkhUtxoList...)

	// P2WPKH
	p2wpkhUtxoList, err := ec.getUtxosForWitnessPubKeyHash(pubKey)
	if err != nil {
		return utxoList, err
	}
	fmt.Printf("found %d P2WPKH utxos\n", len(p2wpkhUtxoList))
	utxoList = append(utxoList, p2wpkhUtxoList...)

	return utxoList, nil
}

type utxoInfo struct {
	utxo    *wallet.Utxo    // utxo we will spend with spend key imported
	address btcutil.Address // address & address type that electrumX stores
}

// ImportAndSweep imports privkeys from other wallets and builds a transaction that
// sweeps their funds into our wallet.
func (ec *BtcElectrumClient) ImportAndSweep(importedKeys []string) error {
	w := ec.GetWallet()
	if w == nil {
		return ErrNoWallet
	}
	if len(importedKeys) <= 0 {
		return errors.New("no keys")
	}
	for _, k := range importedKeys {
		fmt.Printf("trying: %s\n", k)

		wif, err := btcutil.DecodeWIF(k)
		if err != nil {
			fmt.Printf("warning cannot decode WIF from string: %s\n", k)
			continue
		}
		pubKey := wif.SerializePubKey()

		ec.getUtxos(pubKey)
	}

	return nil
}
