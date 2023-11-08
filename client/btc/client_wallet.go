package btc

import (
	"encoding/hex"
	"errors"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
)

// Here is the client interface between the node & wallet for transaction
// broadcast, address monitoring & status of wallet 'scripthashes'

// https://electrumx.readthedocs.io/en/latest/protocol-basics.html

func init() {
	addrToScriptHash = make(map[string]*subscription)
}

// AddrToScripthash takes a bech or legacy bitcoin address and makes an electrum
// 1.4 protocol 'scripthash'
func AddrToScripthash(addr string, network *chaincfg.Params) (string, error) {
	var scripthash string
	if len(addr) <= 0 {
		return "", errors.New("zero length string")
	}

	revBytes := func(b []byte) []byte {
		size := len(b)
		buf := make([]byte, size)
		var i int
		for i = 0; i < size; i++ {
			buf[i] = b[size-i-1]
		}
		return buf
	}

	address, err := btcutil.DecodeAddress(addr, network)
	if err != nil {
		return "", err
	}

	pkscript, err := txscript.PayToAddrScript(address)
	if err != nil {
		return "", err
	}

	pkScriptHashBytes := chainhash.HashB(pkscript)
	revScriptHashBytes := revBytes(pkScriptHashBytes)
	scripthash = hex.EncodeToString(revScriptHashBytes)

	return scripthash, nil
}

type history struct {
	height int32
	txHash string
}

type subscription struct {
	scriptHash  string
	historyList []*history
}

var (
	addrToScriptHash     map[string]*subscription
	ErrAlreadySubscribed error = errors.New("addr already subscribed")
)

// alreadySubscribed checks if this addr is already subscribed
func alreadySubscribed(addr string) bool {
	_, exists := addrToScriptHash[addr]
	return exists
}

// SubscribeAddressNotify subscribes to notifications for an address
func (ec *BtcElectrumClient) SubscribeAddressNotify(addr string) error {
	if alreadySubscribed(addr) {
		return ErrAlreadySubscribed
	}

	// subscribe

	return nil
}

// UnsubscribeAddressNotify unsubscribes from notifications for an address
func (ec *BtcElectrumClient) UnsubscribeAddressNotify(addr string) {
	if !alreadySubscribed(addr) {
		return
	}

	// unsubscribe
}

// Broadcast sends a transaction to the server for broadcast on the bitcoin
// network
func (ec *BtcElectrumClient) Broadcast(rawTx string) (string, error) {
	return ec.GetNode().Broadcast(rawTx)
}
