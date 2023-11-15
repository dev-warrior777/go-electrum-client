package client

import (
	"github.com/btcsuite/btcd/btcutil"
	"github.com/dev-warrior777/go-electrum-client/electrumx"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

type NodeType int

const (
	SingleNode      NodeType = iota
	MultiNode       NodeType = 1
	LOOKAHEADWINDOW          = 30
)

type ElectrumClient interface {
	GetConfig() *ClientConfig
	GetWallet() wallet.ElectrumWallet
	GetNode() electrumx.ElectrumXNode
	//
	CreateNode(nodeType NodeType)
	//
	SyncClientHeaders() error
	SubscribeClientHeaders() error
	//
	CreateWallet(pw string) error
	RecreateWallet(pw, mnenomic string) error
	LoadWallet(pw string) error
	//
	Broadcast(rawTx string) (string, error)
	//
	SyncWallet() error
	SubscribeAddressNotify(address btcutil.Address) error
	UnsubscribeAddressNotify(address btcutil.Address)
	//...
}
