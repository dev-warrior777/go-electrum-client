package btc

import (
	"context"
	"encoding/hex"

	"github.com/dev-warrior777/go-electrum-client/electrumx"
)

// Note: The below are walletless server queries, results are not checked by SPV.

// Return the raw transaction bytes for a txid.
func (ec *BtcElectrumClient) GetRawTransaction(ctx context.Context, txid string) ([]byte, error) {
	node := ec.GetNode()
	if node == nil {
		return nil, ErrNoNode
	}
	txStr, err := node.GetRawTransaction(ctx, txid)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(txStr)
}

// Return the transaction info for a txid.
func (ec *BtcElectrumClient) GetTransaction(ctx context.Context, txid string) (*electrumx.GetTransactionResult, error) {
	node := ec.GetNode()
	if node == nil {
		return nil, ErrNoNode
	}
	res, err := node.GetTransaction(ctx, txid)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Return the transaction history of any address.
func (ec *BtcElectrumClient) GetAddressHistory(ctx context.Context, addr string) (electrumx.HistoryResult, error) {
	node := ec.GetNode()
	if node == nil {
		return nil, ErrNoNode
	}
	scripthash, err := addrToElectrumScripthash(addr, ec.GetConfig().Params)
	if err != nil {
		return nil, err
	}
	return node.GetHistory(ctx, scripthash)
}

func (ec *BtcElectrumClient) GetAddressUnspent(ctx context.Context, addr string) (electrumx.ListUnspentResult, error) {
	node := ec.GetNode()
	if node == nil {
		return nil, ErrNoNode
	}
	scripthash, err := addrToElectrumScripthash(addr, ec.GetConfig().Params)
	if err != nil {
		return nil, err
	}
	return node.GetListUnspent(ctx, scripthash)
}
