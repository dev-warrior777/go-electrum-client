package client

import (
	"github.com/dev-warrior777/go-electrum-client/electrumx"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

type ElectrumClient interface {
	GetConfig() *ClientConfig
	GetWallet() wallet.ElectrumWallet
	GetNode() electrumx.ElectrumXNode
	//
	CreateWallet(pw string) error
	RecreateElectrumWallet(pw, mnenomic string) error
	LoadWallet(pw string) error
	//
	CreateNode()
	SyncHeaders() error
}
