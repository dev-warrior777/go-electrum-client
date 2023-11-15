package btc

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
)

// Here is the client interface between the node & wallet for transaction
// broadcast, address monitoring & status of wallet 'scripthashes'

// https://electrumx.readthedocs.io/en/latest/protocol-basics.html

// It can get confusing. 'scripthash' is an electrum value.

var (
	ErrAlreadySubscribed error = errors.New("addr already subscribed")
)

// alreadySubscribed checks if this addr is already subscribed
func (ec *BtcElectrumClient) alreadySubscribed(address btcutil.Address) bool {
	_, exists := ec.walletSynchronizer.scriptHashToAddr[address]
	return exists
}

func (ec *BtcElectrumClient) SyncWallet() error {

	// - get all receive addresses in wallet
	addresses := ec.GetWallet().ListAddresses()

	// for each
	//   - subscribe for scripthash notifications
	//   - on sub the return is hash of all known history known to server
	//   - get the prev stored history for the address from db if any
	//   - hash prev history and compare to what the server sends back
	//   - if different get the up to date

	for _, address := range addresses {

		// fun with dick & jane
		script := address.ScriptAddress()
		s := hex.EncodeToString(script)
		fmt.Println("ScriptAddress", s)
		segwitAddress, swerr := btcutil.NewAddressWitnessPubKeyHash(script, ec.GetConfig().Params)
		if swerr != nil {
			fmt.Println(swerr)
			continue
		}
		fmt.Println("segwitAddress", segwitAddress.String())
		fmt.Println("segwitAddress", segwitAddress.EncodeAddress())
		address = segwitAddress
		// end fun

		err := ec.SubscribeAddressNotify(address)
		if err != nil {
			return err
		}

	}

	// start goroutine to listen for scripthash status change notifications arriving

	return nil
}

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
	fmt.Println("unsubscribed scripthash")
}

// Broadcast sends a transaction to the server for broadcast on the bitcoin
// network
func (ec *BtcElectrumClient) Broadcast(rawTx string) (string, error) {
	return ec.GetNode().Broadcast(rawTx)
}
