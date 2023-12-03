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
	//
	// Simple RPC server for test only; not production
	RPCServe()
	// Small subset of electrum python console-like methods
	Tip() (int64, bool)
	Spend(amount int64, toAddress string, feeLevel wallet.FeeLevel, broadcast bool) (string, string, error)
	Broadcast(rawTx string) (string, error)
	ListUnspent() ([]wallet.Utxo, error)
	//...
}
