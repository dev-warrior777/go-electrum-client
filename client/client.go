package client

///////////////////////////////// Client interface ///////////////////////////
//
//	architecture
//
//	   Client
//
//	     /\
//	 (controller)
//	   /    \
//	  /      \
//	 /        \
//
// Wallet     Node
//
// The client interface describes the behaviors of the client controller.
// It is implemented for each coin asset client.

import (
	"context"

	"github.com/btcsuite/btcd/wire"
	"github.com/dev-warrior777/go-electrum-client/electrumx"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

type NodeType int

const (
	// ElectrumX Server(s)
	SingleNode NodeType = iota
	MultiNode  NodeType = 1
)

const (
	// Electrum Wallet
	GAP_LIMIT = 10
)

type BroadcastParams struct {
	Tx          *wire.MsgTx
	ChangeIndex int
	ownVouts    []int
	//...
}

type ElectrumClient interface {
	Start(ctx context.Context) error
	Stop()
	//
	GetConfig() *ClientConfig
	GetWallet() wallet.ElectrumWallet
	GetNode() electrumx.ElectrumXNode
	//
	RegisterTipChangeNotify() (<-chan int64, error)
	UnregisterTipChangeNotify()
	//
	CreateWallet(pw string) error
	RecreateWallet(pw, mnenomic string) error
	LoadWallet(pw string) error
	//
	SyncWallet() error
	RescanWallet() error
	ImportAndSweep(wifKeyPairs []string) error
	//
	CloseWallet()
	//
	// Small subset of electrum methods
	Tip() (int64, bool)
	GetBlockHeader(height int64) *wire.BlockHeader
	GetBlockHeaders(startHeight, count int64) ([]*wire.BlockHeader, error)
	Spend(pw string, amount int64, toAddress string, feeLevel wallet.FeeLevel) (int, string, string, error)
	Broadcast(*BroadcastParams) (string, error)
	ListUnspent() ([]wallet.Utxo, error)
	FreezeUTXO(txid string, out uint32) error
	UnfreezeUTXO(txid string, out uint32) error
	UnusedAddress() (string, error)
	ChangeAddress() (string, error)
	Balance() (int64, int64, error)
	FeeRate(confTarget int64) (int64, error)

	//pass thru
	GetTransaction(txid string) (*electrumx.GetTransactionResult, error)
	GetRawTransaction(txid string) ([]byte, error)
	GetAddressHistory(addr string) (electrumx.HistoryResult, error)
	//...

}
