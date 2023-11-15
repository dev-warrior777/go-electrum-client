package btc

import (
	"encoding/hex"
	"strings"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/dev-warrior777/go-electrum-client/client"
)

// historyToStatusHash hashes together the stored history list for a subscription
func historyToStatusHash(s *subscription) string {
	if s == nil || len(s.historyList) == 0 {
		return ""
	}
	sb := strings.Builder{}
	for _, h := range s.historyList {
		sb.WriteString(h.txHash)
		sb.WriteString(":")
		sb.WriteString(string(h.height))
		sb.WriteString(":")
	}
	// history hash as returned from 'blockchain.scripthash.subscribe'
	return string(chainhash.HashB([]byte(sb.String())))
}

type history struct {
	height int32
	txHash string
}

type subscription struct {
	address     btcutil.Address
	historyList []*history
}

type AddressSynchronizer struct {
	scriptHashToAddr map[btcutil.Address]*subscription
	subQueue         []string
	network          *chaincfg.Params
}

func NewWalletSychronizer(cfg *client.ClientConfig) *AddressSynchronizer {
	as := AddressSynchronizer{
		scriptHashToAddr: make(map[btcutil.Address]*subscription, 60),
		subQueue:         make([]string, 0, 60),
		network:          cfg.Params,
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
