package btc

import (
	"fmt"

	"github.com/dev-warrior777/go-electrum-client/wallet"
)

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

	watchedScripts, err := ec.GetWallet().ListSubscribeScripts()
	if err != nil {
		return err
	}

	for _, watchedScript := range watchedScripts {
		address, err := ec.GetWallet().ScriptToAddress(watchedScript)
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
// Python console subset
////////////////////////

// Broadcast sends a transaction to the server for broadcast on the bitcoin
// network
func (ec *BtcElectrumClient) Broadcast(rawTx string) (string, error) {
	return ec.GetNode().Broadcast(rawTx)
}
