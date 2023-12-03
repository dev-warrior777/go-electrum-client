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

var ErrNoWallet error = errors.New("no wallet")

// Here is the client interface between the node & wallet for transaction
// broadcast and wallet synchronize

// devdbg: just one known wallet address
func (ec *BtcElectrumClient) SyncWallet() error {

	address, err := ec.GetWallet().GetUnusedAddress(wallet.RECEIVING)
	if err != nil {
		return err
	}

	payToAddrScript, err := ec.GetWallet().AddressToScript(address)
	if err != nil {
		return err
	}

	err = ec.GetWallet().AddSubscribeScript(payToAddrScript)
	if err != nil {
		return err
	}

	//..................

	subscribeScripts, err := ec.GetWallet().ListSubscribeScripts()
	if err != nil {
		return err
	}

	for _, subscribeScript := range subscribeScripts {
		address, err := ec.GetWallet().ScriptToAddress(subscribeScript)
		if err != nil {
			return err
		}
		fmt.Println(address.String())

		status, err := ec.SubscribeAddressNotify(address)
		if err != nil {
			return err
		}
		if status == "" {
			fmt.Println("no history for this address .. yet")
			continue
		}

		// grab all address history to date for this address
		history, err := ec.GetAddressHistoryFromNode(address)
		if err != nil {
			return err
		}
		// dumpHistory(address, history)

		// update wallet txstore if needed
		ec.addTxHistoryToWallet(history)
	}

	// start goroutine to listen for scripthash status change notifications arriving
	err = ec.addressStatusNotify()
	if err != nil {
		return err
	}

	return nil
}

//-------------------------------------------------
// func (ec *BtcElectrumClient) SyncWallet() error {

// 	// - get all watched receive/our change addresses in wallet

// 	// for each
// 	//   - subscribe for scripthash notifications
// 	//   - on sub the return is hash of all known history known to server
// 	//   - get the prev stored history for the address from db if any
// 	//   - hash prev history and compare to what the server sends back
// 	//   - if different get the up to date history list of txid:height
// 	//     - update db

// 	for _, address := range addresses {

// 		// // fun with dick & jane
// 		// script := address.ScriptAddress()
// 		// s := hex.EncodeToString(script)
// 		// fmt.Println("ScriptAddress", s)
// 		// segwitAddress, swerr := btcutil.NewAddressWitnessPubKeyHash(script, ec.GetConfig().Params)
// 		// if swerr != nil {
// 		// 	fmt.Println(swerr)
// 		// 	continue
// 		// }
// 		// fmt.Println("segwitAddress", segwitAddress.String())
// 		// fmt.Println("segwitAddress", segwitAddress.EncodeAddress())
// 		// address = segwitAddress
// 		// // end fun

// 		err := ec.SubscribeAddressNotify(address)
// 		if err != nil {
// 			return err
// 		}
// 		// AddWatchedScript adds the pkscript to db. If it already exists this
// 		// is a no-op
// 		err = ec.GetWallet().AddWatchedScript(address.ScriptAddress())
// 		if err != nil {
// 			return err
// 		}
// 	}

// 	// start goroutine to listen for scripthash status change notifications arriving

// 	return nil
// }
//------------------------------------------

//////////////////////////////////////////////////////////////////////////////
// Python console-like subset
/////////////////////////////

// Spend tries to create a new transaction to pay amount from the wallet to
// toAddress. It returns Tx & Txid as hex strings. Optionally it broadcasts
// to the network via electrumX.
func (ec *BtcElectrumClient) Spend(
	amount int64,
	toAddress string,
	feeLevel wallet.FeeLevel,
	broadcast bool) (string, string, error) {

	w := ec.GetWallet()
	if w == nil {
		return "", "", ErrNoWallet
	}

	address, err := btcutil.DecodeAddress(toAddress, ec.ClientConfig.Params)
	if err != nil {
		return "", "", err
	}

	wireTx, err := w.Spend(amount, address, feeLevel, false)
	if err != nil {
		return "", "", err
	}

	txidHex := wireTx.TxHash().String()

	// len 0, cap >= serial size
	b := make([]byte, 0, wireTx.SerializeSize())
	buf := bytes.NewBuffer(b)
	err = wireTx.BtcEncode(buf, 0, wire.WitnessEncoding)
	if err != nil {
		return "", "", err
	}
	rawTxHex := hex.EncodeToString(buf.Bytes())

	if !broadcast {
		return rawTxHex, txidHex, nil
	}

	txidHexBroadcast, err := ec.Broadcast(rawTxHex)
	if err != nil {
		return "", "", err
	}

	if txidHex != txidHexBroadcast {
		return "", "", errors.New("broadcast return error - txids inconsistent")
	}

	return rawTxHex, txidHex, nil
}

// Broadcast sends a transaction to the server for broadcast on the bitcoin
// network. It returns txid as a string.
func (ec *BtcElectrumClient) Broadcast(rawTx string) (string, error) {
	return ec.GetNode().Broadcast(rawTx)
}

// ListUnspent
func (ec *BtcElectrumClient) ListUnspent() ([]wallet.Utxo, error) {
	w := ec.GetWallet()
	if w == nil {
		return nil, ErrNoWallet
	}
	return w.ListUnspent()
}
