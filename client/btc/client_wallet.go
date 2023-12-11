package btc

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/wire"
	"github.com/dev-warrior777/go-electrum-client/client"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

var ErrNoWallet error = errors.New("no wallet")
var ErrNoNode error = errors.New("no node")

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

		// - get all subscribed receive/change addresses in wallet
		//
		// for each
		//   - subscribe for scripthash notifications
		//   - on sub the return is hash of all known history known to server
		//   - get the up to date history list of txid:height, if any
		//     - update db

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
// toAddress. It returns Tx & Txid as hex strings. The client needs to know
// the change address so it can set up notify from ElectrumX.
// to the network via electrumX.
func (ec *BtcElectrumClient) Spend(
	pw string,
	amount int64,
	toAddress string,
	feeLevel wallet.FeeLevel) (int, string, string, error) {

	w := ec.GetWallet()
	if w == nil {
		return -1, "", "", ErrNoWallet
	}
	w.UpdateTip(ec.Tip())

	address, err := btcutil.DecodeAddress(toAddress, ec.ClientConfig.Params)
	if err != nil {
		return -1, "", "", err
	}

	changeIndex, wireTx, err := w.Spend(pw, amount, address, feeLevel, false)
	if err != nil {
		return -1, "", "", err
	}

	txidHex := wireTx.TxHash().String()

	// len 0, cap >= serial size
	b := make([]byte, 0, wireTx.SerializeSize())
	buf := bytes.NewBuffer(b)
	err = wireTx.BtcEncode(buf, 0, wire.WitnessEncoding)
	if err != nil {
		return -1, "", "", err
	}
	rawTxHex := hex.EncodeToString(buf.Bytes())

	return changeIndex, rawTxHex, txidHex, nil
}

// ExternalBroadcast sends a transaction to the server for broadcast on the bitcoin
// network. It returns txid as a string. It is not part of the ElectrumClient
// interface.
func (ec *BtcElectrumClient) ExternalBroadcast(rawTx string, changeIndex int) (string, error) {
	txid, err := ec.GetNode().Broadcast(rawTx)
	if err != nil {
		return "", err
	}
	return txid, nil
}

// Broadcast sends a transaction to the server for broadcast on the bitcoin
// network. It returns txid as a string.
func (ec *BtcElectrumClient) Broadcast(bc *client.BroadcastParams) (string, error) {
	node := ec.GetNode()
	if node == nil {
		return "", ErrNoNode
	}
	if bc.Tx == nil {
		return "", errors.New("nil Tx")
	}

	// serialize tx
	b := make([]byte, bc.Tx.SerializeSize())
	bb := bytes.NewBuffer(b)
	rawTx := bb.Bytes()

	// check change index is valid
	if bc.ChangeIndex >= 0 {
		fmt.Println("change output index", bc.ChangeIndex)
		txOuts := bc.Tx.TxOut
		if len(txOuts) < bc.ChangeIndex+1 {
			return "", errors.New("invalid change index")
		}
	}

	// Send tx to ElectrumX for broadcasting to the bitcoin network
	rawTxStr := hex.EncodeToString(rawTx)
	txid, err := node.Broadcast(rawTxStr)
	if err != nil {
		return "", err
	}

	// Subscribe any addresses we might be interested in. This should also add
	// the containing tx to the wallet. In particular we almost always have a
	// change script address to watch paying back to our wallet after tx mined.

	change := bc.Tx.TxOut[bc.ChangeIndex]
	scripthash := ec.walletSynchronizer.pkScriptToElectrumScripthash(change.PkScript)
	res, err := node.SubscribeScripthashNotify(scripthash)
	if err != nil {
		return "", err
	}
	if res == nil {
		return "", errors.New("empty result")
	}

	return txid, nil
}

// ListUnspent
func (ec *BtcElectrumClient) ListUnspent() ([]wallet.Utxo, error) {
	w := ec.GetWallet()
	if w == nil {
		return nil, ErrNoWallet
	}
	return w.ListUnspent()
}
