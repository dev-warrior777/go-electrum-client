package client

import (
	"github.com/dev-warrior777/go-electrum-client/electrumx"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

type ElectrumClient interface {
	CreateWallet(pw string) error
	RecreateElectrumWallet(pw, mnenomic string) error
	LoadWallet(pw string) error
	//
	Config() *Config
	Wallet() wallet.ElectrumWallet
	Node() electrumx.ElectrumXNode
}
