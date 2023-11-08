package client

import (
	"github.com/dev-warrior777/go-electrum-client/electrumx"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

type NodeType int

const (
	SingleNode NodeType = iota
	MultiNode  NodeType = 1
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
	RecreateElectrumWallet(pw, mnenomic string) error
	LoadWallet(pw string) error
	//
	Broadcast(rawTx string) (string, error)
	SubscribeAddressNotify(addr string) error
	UnsubscribeAddressNotify(addr string)
	//...
}
