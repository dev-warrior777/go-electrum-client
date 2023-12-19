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
	"github.com/btcsuite/btcd/btcutil"
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
	AGEDTX    = 3
)

type BroadcastParams struct {
	Tx              *wire.MsgTx
	PrevScripts     [][]byte
	PrevInputValues []btcutil.Amount
	TotalInput      btcutil.Amount
	ChangeIndex     int // negative if no change
	RedeemScripts   [][]byte
}

type ElectrumClient interface {
	GetConfig() *ClientConfig
	GetWallet() wallet.ElectrumWallet
	GetNode() electrumx.ElectrumXNode
	//
	CreateNode(nodeType NodeType)
	//
	SyncHeaders() error
	SubscribeClientHeaders() error
	//
	CreateWallet(pw string) error
	RecreateWallet(pw, mnenomic string) error
	LoadWallet(pw string) error
	//
	SyncWallet() error
	Close()
	//
	// Simple RPC server for test only; not production
	RPCServe() error
	// Small subset of electrum python console-like methods
	Tip() (int64, bool)
	Spend(pw string, amount int64, toAddress string, feeLevel wallet.FeeLevel) (int, string, string, error)
	Broadcast(*BroadcastParams) (string, error)
	ListUnspent() ([]wallet.Utxo, error)
	UnusedAddress() (string, error)
	Balance() (int64, int64, error)
	//...
}
