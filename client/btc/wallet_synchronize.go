package btc

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

	"github.com/dev-warrior777/go-electrum-client/electrumx"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

// Here is the client interface between the node & wallet for monitoring the status
// of wallet 'scripthashes'
//
// https://electrumx.readthedocs.io/en/latest/protocol-basics.html
//
// It can get confusing! Here 'scripthash' is an electrum value. But the
// ScriptHash from (btcutl.Address).SciptHash() is the normal RIPEMD160
// hash.
//
// An electrum scripthash is the full output payment script which is then
// sha256 hashed. The result has bytes reversed for network send. It is sent
// to ElectrumX as a string.

// // historyToStatusHash hashes together the stored history list for a subscription
// func historyToStatusHash(history electrumx.HistoryResult) string {
// 	if len(history) == 0 {
// 		return ""
// 	}
// 	sb := strings.Builder{}
// 	for _, h := range history {
// 		sb.WriteString(h.TxHash)
// 		sb.WriteString(":")
// 		sb.WriteString(fmt.Sprintf("%d", h.Height))
// 		sb.WriteString(":")
// 	}
// 	return hex.EncodeToString(chainhash.HashB([]byte(sb.String())))
// }

var ErrNoSubscriptionFoundInDb = errors.New("no subscription found in db")

func (ec *BtcElectrumClient) addSubscription(subscription *wallet.Subscription) error {
	w := ec.GetWallet()
	if w == nil {
		return ErrNoWallet
	}
	return w.AddSubscription(subscription)
}

func (ec *BtcElectrumClient) getSubscriptionForScripthash(scripthash string) (*wallet.Subscription, error) {
	w := ec.GetWallet()
	if w == nil {
		return nil, ErrNoWallet
	}
	subscription, err := w.GetSubscriptionForElectrumScripthash(scripthash)
	if err != nil {
		return nil, err
	}
	if subscription == nil {
		return nil, ErrNoSubscriptionFoundInDb
	}
	return subscription, nil
}

func (ec *BtcElectrumClient) getSubscription(scriptPubKey string) (*wallet.Subscription, error) {
	w := ec.GetWallet()
	if w == nil {
		return nil, ErrNoWallet
	}
	subscription, err := w.GetSubscription(scriptPubKey)
	if err != nil {
		return nil, err
	}
	if subscription == nil {
		return nil, ErrNoSubscriptionFoundInDb
	}
	return subscription, nil
}

func (ec *BtcElectrumClient) isSubscribed(pkScript string) (bool, error) {
	w := ec.GetWallet()
	if w == nil {
		return false, ErrNoWallet
	}
	subscription, err := w.GetSubscription(pkScript)
	if err != nil {
		return false, err
	}
	if subscription == nil {
		return false, nil
	}
	return true, nil
}

func (ec *BtcElectrumClient) removeSubscription(pkScript string) error {
	w := ec.GetWallet()
	if w == nil {
		return ErrNoWallet
	}
	w.RemoveSubscription(pkScript)
	return nil
}

// pkScriptToElectrumScripthash takes a public key script and makes an electrum
// 1.4 protocol 'scripthash'
func pkScriptToElectrumScripthash(pkScript []byte) string {
	revBytes := func(b []byte) []byte {
		size := len(b)
		buf := make([]byte, size)
		var i int
		for i = 0; i < size; i++ {
			buf[i] = b[size-i-1]
		}
		return buf
	}
	fmt.Println("pkScript", hex.EncodeToString(pkScript), " before electrum extra hashing")
	pkScriptHashBytes := chainhash.HashB(pkScript)
	revScriptHashBytes := revBytes(pkScriptHashBytes)
	return hex.EncodeToString(revScriptHashBytes)
}

// addrToScripthash takes a btcutil.Address and makes an electrum 1.4 protocol 'scripthash'
func addressToElectrumScripthash(address btcutil.Address, network *chaincfg.Params) (string, error) {
	pkScript, err := txscript.PayToAddrScript(address)
	if err != nil {
		return "", err
	}
	return pkScriptToElectrumScripthash(pkScript), nil
}

// addrToElectrumScripthash takes a bech or legacy bitcoin address and makes an electrum
// 1.4 protocol 'scripthash'
func addrToElectrumScripthash(addr string, network *chaincfg.Params) (string, error) {
	address, err := btcutil.DecodeAddress(addr, network)
	if err != nil {
		return "", err
	}
	return addressToElectrumScripthash(address, network)
}

//////////////////////////////////////////////////////////////////////////////
// wallet <-> client <-> node
/////////////////////////////

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

			case status := <-scripthashNotifyCh:
				fmt.Printf("\n\n%s\n", "----------------------------------------")
				fmt.Println("<-scripthashNotifyCh - # items left in buffer", len(scripthashNotifyCh))
				fmt.Println("scripthash notify")
				if status == nil {
					fmt.Println("status is nil, ignoring...", status)
					continue
				}
				fmt.Println("Scripthash", status.Scripthash)
				fmt.Println("Status", status.Status)
				if status.Status == "" {
					fmt.Println("status.Status is null", status.Status, " no history yet; ignoring...")
					continue
				}
				// is status same as last status?
				sub, err := ec.getSubscriptionForScripthash(status.Scripthash)
				if err != nil {
					panic(err) ////////////// no rows in result set ////////////////// 2
				}
				if sub == nil {
					panic("no subscription for subscribed scripthash")
				}

				// get scripthash history
				history, err := ec.GetAddressHistoryFromNode(sub)
				if err != nil {
					continue
				}
				ec.dumpHistory(sub, history)

				// update wallet txstore
				ec.addTxHistoryToWallet(history)
			}
		}
	}()
	// serve until done
	return nil
}

// SubscribeAddressNotify subscribes to notifications from ElectrumX for a public
// key script address. It also adds the new subscription to the wallet database.
// It returns a subscribe status which is the hash of all address history known
// to the electrumX server and can be zero length string if the subscription is
// new and has no history.
func (ec *BtcElectrumClient) SubscribeAddressNotify(newSub *wallet.Subscription) (string, error) {
	node := ec.GetNode()
	if node == nil {
		return "", ErrNoNode
	}
	subscribed, err := ec.isSubscribed(newSub.PkScript)
	if err != nil {
		return "", err
	}

	// subscribe to node and wallet database
	res, err := node.SubscribeScripthashNotify(newSub.ElectrumScripthash)
	if err != nil {
		return "", err
	}
	if res == nil {
		return "", errors.New("empty result")
	}
	// wallet
	if !subscribed {
		ec.addSubscription(newSub)
	}

	fmt.Println("Subscribed scripthash", res.Scripthash, " status:", res.Status)

	return res.Status, nil
}

// UnsubscribeAddressNotify unsubscribes from notifications for an address
// and removes the subscription from the wallet database
func (ec *BtcElectrumClient) UnsubscribeAddressNotify(pkScript string) {
	node := ec.GetNode()
	if node == nil {
		return
	}
	subscription, err := ec.getSubscription(pkScript)
	if err != nil || subscription == nil {
		fmt.Println("not subscribed or db error")
		return
	}

	// unsubscribe from node and wallet database
	node.UnsubscribeScripthashNotify(subscription.ElectrumScripthash)
	err = ec.removeSubscription(pkScript)
	if err != nil {
		return
	}
	fmt.Println("unsubscribed scripthash")
}

func (ec *BtcElectrumClient) GetAddressHistoryFromNode(subscription *wallet.Subscription) (electrumx.HistoryResult, error) {
	node := ec.GetNode()
	if node == nil {
		return nil, ErrNoNode
	}
	res, err := ec.GetNode().GetHistory(subscription.ElectrumScripthash)
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		fmt.Println("empty history result for: ", subscription.PkScript)
		return nil, nil
	}

	return res, nil
}

func (ec *BtcElectrumClient) GetRawTransactionFromNode(txid string) (*wire.MsgTx, time.Time, error) {
	node := ec.GetNode()
	if node == nil {
		return nil, time.Time{}, ErrNoNode
	}
	txres, err := node.GetRawTransaction(txid)
	if err != nil {
		return nil, time.Time{}, err
	}
	b, err := hex.DecodeString(txres)
	if err != nil {
		return nil, time.Time{}, err
	}
	hexBuf := bytes.NewBuffer(b)
	var msgTx wire.MsgTx = wire.MsgTx{Version: 1}
	err = msgTx.BtcDecode(hexBuf, 1, wire.WitnessEncoding) // careful here witness!
	if err != nil {
		return nil, time.Time{}, err
	}
	txTime := time.Now()
	return &msgTx, txTime, nil
}

func (ec *BtcElectrumClient) addTxHistoryToWallet(history electrumx.HistoryResult) {
	for _, h := range history {
		txid, err := hex.DecodeString(h.TxHash)
		if err != nil {
			continue
		}
		txhash, err := chainhash.NewHash(txid)
		if err != nil {
			continue
		}
		fmt.Println(txhash.String())

		// does wallet already has a confirmed transaction?
		walletHasTx := ec.GetWallet().HasTransaction(*txhash)
		fmt.Println("walletHasTx", walletHasTx)
		if walletHasTx && h.Height > 0 {
			fmt.Println("** already got confirmed tx", txid)
			continue
		}

		// add or update the wallet transaction
		msgTx, txtime, err := ec.GetRawTransactionFromNode(h.TxHash)
		if err != nil {
			continue
		}
		fmt.Println(msgTx.TxHash().String(), txtime)
		fmt.Println("adding transaction", h.TxHash, h.Height, h.Fee)
		err = ec.GetWallet().AddTransaction(msgTx, h.Height, txtime)
		if err != nil {
			fmt.Println(err)
			continue
		}
	}
}

func (ec *BtcElectrumClient) updateWalletTip() {
	w := ec.GetWallet()
	if w != nil {
		w.UpdateTip(ec.Tip())
	}
}

// //////////////////////////
// dbg dump
// /////////
func (ec *BtcElectrumClient) pkScriptToAddressPubkeyHash(pkScript []byte) (btcutil.Address, string) {
	pks, err := txscript.ParsePkScript(pkScript)
	if err != nil {
		return nil, ""
	}
	apkh, err := pks.Address(ec.GetConfig().Params)
	if err != nil {
		return nil, ""
	}
	return apkh, apkh.String()
}

func (ec *BtcElectrumClient) pkScriptStringToAddressPubkeyHash(pkScriptStr string) (btcutil.Address, string) {
	pkScript, err := hex.DecodeString(pkScriptStr)
	if err != nil {
		return nil, ""
	}
	return ec.pkScriptToAddressPubkeyHash(pkScript)
}

func (ec *BtcElectrumClient) dumpSubscription(title string, sub *wallet.Subscription) {
	fmt.Printf("%s\n PkScript: %s\n ElectrumScriptHash: %s\n Address: %s\n\n",
		title,
		sub.PkScript,
		sub.ElectrumScripthash,
		sub.Address)
}

func (ec *BtcElectrumClient) dumpHistory(sub *wallet.Subscription, history electrumx.HistoryResult) {
	ec.dumpSubscription("Address History for subscription:", sub)
	for _, h := range history {
		fmt.Println(" Height:", h.Height)
		fmt.Println(" TxHash: ", h.TxHash)
		fmt.Println(" Fee: ", h.Fee)
	}
}
