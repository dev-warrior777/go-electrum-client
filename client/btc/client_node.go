package btc

import (
	"encoding/hex"

	"github.com/dev-warrior777/go-electrum-client/electrumx"
)

// Note: The below are walletless server queries, results are not checked by SPV.

// Return the raw transaction bytes for a txid.
func (ec *BtcElectrumClient) GetRawTransaction(txid string) ([]byte, error) {
	node := ec.GetNode()
	if node == nil {
		return nil, ErrNoNode
	}
	txStr, err := node.GetRawTransaction(txid)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(txStr)
}

// Return the transaction info for a txid.
func (ec *BtcElectrumClient) GetTransaction(txid string) (*electrumx.GetTransactionResult, error) {
	node := ec.GetNode()
	if node == nil {
		return nil, ErrNoNode
	}
	res, err := node.GetTransaction(txid)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Return the transaction history of any address.
func (ec *BtcElectrumClient) GetAddressHistory(addr string) (electrumx.HistoryResult, error) {
	node := ec.GetNode()
	if node == nil {
		return nil, ErrNoNode
	}
	scripthash, err := addrToElectrumScripthash(addr, ec.GetConfig().Params)
	if err != nil {
		return nil, err
	}
	return node.GetHistory(scripthash)
}

func (ec *BtcElectrumClient) GetAddressUnspent(addr string) (electrumx.ListUnspentResult, error) {
	node := ec.GetNode()
	if node == nil {
		return nil, ErrNoNode
	}
	scripthash, err := addrToElectrumScripthash(addr, ec.GetConfig().Params)
	if err != nil {
		return nil, err
	}
	return node.GetListUnspent(scripthash)
}
