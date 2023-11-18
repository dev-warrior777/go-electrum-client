package btc

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/dev-warrior777/go-electrum-client/client"
	"github.com/dev-warrior777/go-electrum-client/electrumx"
)

// Here is the client interface between the node & wallet for monitoring the status
// of wallet 'scripthashes'
//
// https://electrumx.readthedocs.io/en/latest/protocol-basics.html
//
// It can get confusing! Here 'scripthash' is an electrum value. But the
// ScriptHash from (btcutl.Address).SciptHash() is the normal RIPEMD160
// hash -- except for SegwitScripthash addresses which are 32 bytes long.
//
// An electrum scripthash is the full output payment script which is then
// sha256 hashed. The result has bytes reversed for network send. It is sent
// to ElectrumX as a string.

// // historyToStatusHash hashes together the stored history list for a subscription
// func historyToStatusHash(s *subscription) string {
// 	if s == nil || len(s.historyList) == 0 {
// 		return ""
// 	}
// 	sb := strings.Builder{}
// 	for _, h := range s.historyList {
// 		sb.WriteString(h.txHash)
// 		sb.WriteString(":")
// 		sb.WriteString(string(h.height))
// 		sb.WriteString(":")
// 	}
// 	// history hash as returned from 'blockchain.scripthash.subscribe'
// 	return string(chainhash.HashB([]byte(sb.String())))
// }

// type history struct {
// 	height int32
// 	txHash string
// }

// We need a mapping both ways
type subscription struct {
	address    btcutil.Address
	scripthash string
	lastStatus string // a future optimization
}

type AddressSynchronizer struct {
	subscriptions map[btcutil.Address]*subscription
	network       *chaincfg.Params
}

func (as *AddressSynchronizer) getAddressForScripthash(scripthash string) btcutil.Address {
	for _, sub := range as.subscriptions {
		if sub.scripthash == scripthash {
			return sub.address
		}
	}
	return nil
}

func (as *AddressSynchronizer) getScripthashForAddress(address btcutil.Address) string {
	sub := as.subscriptions[address]
	return sub.scripthash
}

func (as *AddressSynchronizer) isSubscribed(address btcutil.Address) bool {
	return as.subscriptions[address] != nil
}

func (as *AddressSynchronizer) addSubscription(address btcutil.Address, scripthash string) {
	sub := subscription{
		address:    address,
		scripthash: scripthash,
		lastStatus: "",
	}
	as.subscriptions[address] = &sub
}

func (as *AddressSynchronizer) removeSubscription(address btcutil.Address) {
	if as.subscriptions[address] != nil {
		delete(as.subscriptions, address)
	}
}

func (as *AddressSynchronizer) getSubscriptionForscripthash(scripthash string) *subscription {
	address := as.getAddressForScripthash(scripthash)
	return as.subscriptions[address]
}

func NewWalletSychronizer(cfg *client.ClientConfig) *AddressSynchronizer {
	as := AddressSynchronizer{
		subscriptions: make(map[btcutil.Address]*subscription, client.LOOKAHEADWINDOW*2),
		network:       cfg.Params,
	}
	return &as
}

// addrToScripthash takes a btcutil.Address and makes an electrum 1.4 protocol 'scripthash'
func (as *AddressSynchronizer) addressToElectrumScripthash(address btcutil.Address, network *chaincfg.Params) (string, error) {
	revBytes := func(b []byte) []byte {
		size := len(b)
		buf := make([]byte, size)
		var i int
		for i = 0; i < size; i++ {
			buf[i] = b[size-i-1]
		}
		return buf
	}

	pkscript, err := txscript.PayToAddrScript(address)
	if err != nil {
		return "", err
	}
	fmt.Println("pkscript", hex.EncodeToString(pkscript), " before electrum extra hashing")

	pkScriptHashBytes := chainhash.HashB(pkscript)
	revScriptHashBytes := revBytes(pkScriptHashBytes)
	scripthash := hex.EncodeToString(revScriptHashBytes)

	return scripthash, nil
}

// addrToElectrumScripthash takes a bech or legacy bitcoin address and makes an electrum
// 1.4 protocol 'scripthash'
func (as *AddressSynchronizer) addrToElectrumScripthash(addr string, network *chaincfg.Params) (string, error) {
	address, err := btcutil.DecodeAddress(addr, network)
	if err != nil {
		return "", err
	}
	return as.addressToElectrumScripthash(address, network)
}

//////////////////////////////////////////////////////////////////////////////
// client wallet node
/////////////////////

// addressStatusNotify listens for address status change notifications
func (ec *BtcElectrumClient) addressStatusNotify() error {
	node := ec.GetNode()

	scripthashNotifyCh, err := node.GetScripthashNotify()
	if err != nil {
		return err
	}
	svrCtx := node.GetServerConn().SvrCtx

	go func() {
		fmt.Println("=== Waiting for address change notifications ===")
		for {
			select {

			case <-svrCtx.Done():
				fmt.Println("Server shutdown - scripthash notify")
				node.Stop()
				return

			case <-scripthashNotifyCh:
				for status := range scripthashNotifyCh {
					fmt.Println("scripthash notify")
					fmt.Println("Scripthash", status.Scripthash)
					fmt.Println("Status", status.Status)
					if status.Status == "" {
						continue
					}
					// is status same as last status?
					sub := ec.walletSynchronizer.getSubscriptionForscripthash(status.Scripthash)
					if sub == nil {
						panic("no synchronizer subscription for subscribed scripthash ")
					}
					if sub.lastStatus == status.Status {
						continue
					}
					sub.lastStatus = status.Status

					// get scripthash history
					history, err := ec.GetAddressHistory(sub.address)
					if err != nil {
						continue
					}
					dumpHistory(sub.address, history)

					// update wallet txstore

				}
			}
		}
	}()
	// serve until done
	return nil
}

// alreadySubscribed checks if this address is already subscribed
func (ec *BtcElectrumClient) alreadySubscribed(address btcutil.Address) bool {
	return ec.walletSynchronizer.isSubscribed(address)
}

// SubscribeAddressNotify subscribes to notifications for an address and retrieves
// & stores address history known to the server
func (ec *BtcElectrumClient) SubscribeAddressNotify(address btcutil.Address) error {
	if ec.alreadySubscribed(address) {
		return errors.New("address already subscribed")
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

func (ec *BtcElectrumClient) GetAddressHistory(address btcutil.Address) (electrumx.HistoryResult, error) {
	scripthash, err := ec.walletSynchronizer.addressToElectrumScripthash(
		address, ec.GetConfig().Params)
	if err != nil {
		return nil, err
	}
	res, err := ec.GetNode().GetHistory(scripthash)
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		fmt.Println("empty history result for: ", address.String())
		return nil, nil
	}

	return res, nil
}

func dumpHistory(address btcutil.Address, history electrumx.HistoryResult) {
	fmt.Println("History for address ", address.String())
	for _, h := range history {
		fmt.Println("Height:", h.Height)
		fmt.Println("TxHash: ", h.TxHash)
		fmt.Println("Fee: ", h.Fee)
	}

}
