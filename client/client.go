package client

import (
	"github.com/dev-warrior777/go-electrum-client/electrumx"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

type ElectrumClient interface {
	GetConfig() *ClientConfig
	SetWallet(wallet.ElectrumWallet)
	GetWallet() wallet.ElectrumWallet
	SetNode(electrumx.ElectrumXNode)
	GetNode() electrumx.ElectrumXNode
	//
	CreateWallet(pw string) error
	RecreateElectrumWallet(pw, mnenomic string) error
	LoadWallet(pw string) error
	//
	CreateNode() error
}
