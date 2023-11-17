package btc

import (
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/txscript"
)

// Here is the client interface between the node & wallet for transaction
// broadcast, address monitoring & status of wallet 'scripthashes'

// https://electrumx.readthedocs.io/en/latest/protocol-basics.html

// It can get confusing! Here 'scripthash' is an electrum value. But the
// ScriptHash from (btcutl.Address).SciptHash() is the normal RIPEMD160
// hash -- except for SegwitScripthash addresses which are 32 bytes long.
//
// An electrum scripthash is the full output payment script which is then
// sha256 hashed. The result has bytes reversed for network send. It is sent
// to ElectrumX as a string.

var (
	ErrAlreadySubscribed error = errors.New("address already subscribed")
)

// alreadySubscribed checks if this address is already subscribed
func (ec *BtcElectrumClient) alreadySubscribed(address btcutil.Address) bool {
	return ec.walletSynchronizer.isSubscribed(address)
}

// devdbg
func (ec *BtcElectrumClient) SyncWallet() error {

	// - get all receive addresses in wallet
	addresses := ec.GetWallet().ListAddresses()

	address := addresses[0]

	fmt.Println(address.String())

	err := ec.SubscribeAddressNotify(address)
	if err != nil {
		return err
	}
	script, err := txscript.PayToAddrScript(address)
	if err != nil {
		// possibly non-standard: future
		return err
	}
	// AddWatchedScript adds the pay script to db. If it already exists this
	// is a no-op
	err = ec.GetWallet().AddWatchedScript(script)
	if err != nil {
		return err
	}

	err = ec.GetAddressHistory(address)
	if err != nil {
		return err
	}

	// start goroutine to listen for scripthash status change notifications arriving

	return nil
}

//-------------------------------------------------
// func (ec *BtcElectrumClient) SyncWallet() error {

// 	// - get all receive addresses in wallet
// 	addresses := ec.GetWallet().ListAddresses()

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

// SubscribeAddressNotify subscribes to notifications for an address and retreives
// & stores address history known to the server
func (ec *BtcElectrumClient) SubscribeAddressNotify(address btcutil.Address) error {
	if ec.alreadySubscribed(address) {
		return ErrAlreadySubscribed
	}

	// subscribe
	scripthash, err := ec.walletSynchronizer.addressToElectrumScripthash(
		address, ec.GetConfig().Params)
	if err != nil {
		return err
	}
	res, err := ec.GetNode().SubscribeScripthashNotify(scripthash)
	if err != nil {
		return err
	}
	if res == nil {
		return errors.New("empty result")
	}
	ec.walletSynchronizer.addSubscription(address, scripthash)

	fmt.Println("Subscribed scripthash")
	fmt.Println("Scripthash", res.Scripthash)
	fmt.Println("Status", res.Status)

	return nil
}

// UnsubscribeAddressNotify unsubscribes from notifications for an address
func (ec *BtcElectrumClient) UnsubscribeAddressNotify(address btcutil.Address) {
	if !ec.alreadySubscribed(address) {
		return
	}

	// unsubscribe
	scripthash, err := ec.walletSynchronizer.addressToElectrumScripthash(
		address, ec.GetConfig().Params)
	if err != nil {
		return
	}
	ec.GetNode().UnsubscribeScripthashNotify(scripthash)
	ec.walletSynchronizer.removeSubscription(address)
	fmt.Println("unsubscribed scripthash")
}

func (ec *BtcElectrumClient) GetAddressHistory(address btcutil.Address) error {
	scripthash, err := ec.walletSynchronizer.addressToElectrumScripthash(
		address, ec.GetConfig().Params)
	if err != nil {
		return err
	}
	res, err := ec.GetNode().GetHistory(scripthash)
	if err != nil {
		return err
	}

	if len(res) == 0 {
		fmt.Println("empty history result for: ", address.String())
		return nil
	}
	fmt.Println("History for address ", address.String())
	for _, history := range res {
		fmt.Println("Height:", history.Height)
		fmt.Println("TxHash: ", history.TxHash)
		fmt.Println("Fee: ", history.Fee)
	}

	return nil
}

// Broadcast sends a transaction to the server for broadcast on the bitcoin
// network
func (ec *BtcElectrumClient) Broadcast(rawTx string) (string, error) {
	return ec.GetNode().Broadcast(rawTx)
}
