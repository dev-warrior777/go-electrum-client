package btc

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"

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
// hash.
//
// An electrum scripthash is the full output payment script which is then
// sha256 hashed. The result has bytes reversed for network send. It is sent
// to ElectrumX as a string.

// historyToStatusHash hashes together the stored history list for a subscription
func historyToStatusHash(history electrumx.HistoryResult) string {
	if len(history) == 0 {
		return ""
	}
	sb := strings.Builder{}
	for _, h := range history {
		sb.WriteString(h.TxHash)
		sb.WriteString(":")
		sb.WriteString(fmt.Sprintf("%d", h.Height))
		sb.WriteString(":")
	}
	return hex.EncodeToString(chainhash.HashB([]byte(sb.String())))
}

// We need a mapping both ways
type subscription struct {
	// wallet subscribe watch list public key script. hex string
	pkScript string
	// electrum 1.4 protocol 'scripthash'
	scripthash string
	// a future optimization
	lastStatus string
}

type AddressSynchronizer struct {
	subscriptions    map[string]*subscription
	subscriptionsMtx sync.Mutex
	network          *chaincfg.Params
}

func (as *AddressSynchronizer) getPkScriptForScripthash(scripthash string) string {
	for _, sub := range as.subscriptions {
		if sub.scripthash == scripthash {
			return sub.pkScript
		}
	}
	return ""
}

func (as *AddressSynchronizer) isSubscribed(pkScript string) bool {
	as.subscriptionsMtx.Lock()
	defer as.subscriptionsMtx.Unlock()
	return as.subscriptions[pkScript] != nil
}

func (as *AddressSynchronizer) addSubscription(pkScript string, scripthash string) {
	as.subscriptionsMtx.Lock()
	sub := subscription{
		pkScript:   pkScript,
		scripthash: scripthash,
		lastStatus: "",
	}
	as.subscriptions[pkScript] = &sub
	as.subscriptionsMtx.Unlock()
}

func (as *AddressSynchronizer) removeSubscription(pkScript string) {
	as.subscriptionsMtx.Lock()
	if as.subscriptions[pkScript] != nil {
		delete(as.subscriptions, pkScript)
	}
	as.subscriptionsMtx.Unlock()
}

func (as *AddressSynchronizer) getSubscriptionForScripthash(scripthash string) *subscription {
	pkScript := as.getPkScriptForScripthash(scripthash)
	return as.subscriptions[pkScript]
}

func NewWalletSychronizer(cfg *client.ClientConfig) *AddressSynchronizer {
	as := AddressSynchronizer{
		subscriptions: make(map[string]*subscription, client.GAP_LIMIT*2),
		network:       cfg.Params,
	}
	return &as
}

func (as *AddressSynchronizer) pkScriptStringToElectrumScripthash(pkScript string) (string, error) {
	pkScriptBytes, err := hex.DecodeString(pkScript)
	if err != nil {
		return "", err
	}
	return as.pkScriptToElectrumScripthash(pkScriptBytes), nil
}

// pkScriptToElectrumScripthash takes a public key script and makes an electrum 1.4 protocol 'scripthash'
func (as *AddressSynchronizer) pkScriptToElectrumScripthash(pkScript []byte) string {
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
func (as *AddressSynchronizer) addressToElectrumScripthash(address btcutil.Address, network *chaincfg.Params) (string, error) {
	pkScript, err := txscript.PayToAddrScript(address)
	if err != nil {
		return "", err
	}
	return as.pkScriptToElectrumScripthash(pkScript), nil
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
				sub := ec.walletSynchronizer.getSubscriptionForScripthash(status.Scripthash)
				if sub == nil {
					panic("no synchronizer subscription for subscribed scripthash")
				}
				if sub.lastStatus == status.Status {
					continue
				}
				sub.lastStatus = status.Status

				// get scripthash history
				history, err := ec.GetAddressHistoryFromNode(sub.pkScript)
				if err != nil {
					continue
				}
				ec.dumpHistory(sub.pkScript, history)

				// update wallet txstore
				ec.addTxHistoryToWallet(history)
			}
		}
	}()
	// serve until done
	return nil
}

// alreadySubscribed checks if this public key script address is already subscribed
func (ec *BtcElectrumClient) alreadySubscribed(pkScript string) bool {
	return ec.walletSynchronizer.isSubscribed(pkScript)
}

// SubscribeAddressNotify subscribes to notifications from ElectrumX for a public
// key script address. It returns a subscribe status which is the hash of all
// address history known to the electrumX server and can be zero length string if the subscription is new
// and has no history.
func (ec *BtcElectrumClient) SubscribeAddressNotify(pkScript string) (string, error) {
	node := ec.GetNode()
	if node == nil {
		return "", ErrNoNode
	}
	if ec.alreadySubscribed(pkScript) {
		return "", errors.New("pkScript already subscribed")
	}

	// subscribe
	scripthash, err := ec.walletSynchronizer.pkScriptStringToElectrumScripthash(pkScript)
	if err != nil {
		return "", err
	}
	res, err := node.SubscribeScripthashNotify(scripthash)
	if err != nil {
		return "", err
	}
	if res == nil {
		return "", errors.New("empty result")
	}
	ec.walletSynchronizer.addSubscription(pkScript, scripthash)

	fmt.Println("Subscribed scripthash", res.Scripthash, " status:", res.Status)

	return res.Status, nil
}

// UnsubscribeAddressNotify unsubscribes from notifications for an address
func (ec *BtcElectrumClient) UnsubscribeAddressNotify(pkScript string) {
	node := ec.GetNode()
	if node == nil {
		return
	}
	if !ec.alreadySubscribed(pkScript) {
		return
	}

	// unsubscribe
	scripthash, err := ec.walletSynchronizer.pkScriptStringToElectrumScripthash(pkScript)
	if err != nil {
		return
	}
	node.UnsubscribeScripthashNotify(scripthash)
	ec.walletSynchronizer.removeSubscription(pkScript)
	fmt.Println("unsubscribed scripthash")
}

func (ec *BtcElectrumClient) GetAddressHistoryFromNode(pkScript string) (electrumx.HistoryResult, error) {
	node := ec.GetNode()
	if node == nil {
		return nil, ErrNoNode
	}
	scripthash, err := ec.walletSynchronizer.pkScriptStringToElectrumScripthash(pkScript)
	if err != nil {
		return nil, err
	}
	res, err := ec.GetNode().GetHistory(scripthash)
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		fmt.Println("empty history result for: ", pkScript)
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
///////////

func (ec *BtcElectrumClient) pkScriptStringToAddressPubkeyHash(pkScriptStr string) (btcutil.Address, string) {
	pkScript, err := hex.DecodeString(pkScriptStr)
	if err != nil {
		return nil, ""
	}
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

func (ec *BtcElectrumClient) dumpHistory(pkScript string, history electrumx.HistoryResult) {
	_, apkhs := ec.pkScriptStringToAddressPubkeyHash(pkScript)
	fmt.Println("Address Hsitory for script address", pkScript, apkhs)
	for _, h := range history {
		fmt.Println("Height:", h.Height)
		fmt.Println("TxHash: ", h.TxHash)
		fmt.Println("Fee: ", h.Fee)
	}
}
