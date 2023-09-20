package btc

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/dev-warrior777/go-electrum-client/wallet"
	"github.com/dev-warrior777/go-electrum-client/wallet/db"
)

// BtcElectrumClient
type BtcElectrumClient struct {
	config       *wallet.Config
	wallet       wallet.ElectrumWallet
	chainManager *wallet.ChainManager
}

func NewBtcElectrumClient(cfg *wallet.Config) *BtcElectrumClient {
	manager := newchainManager(cfg)

	// Select wallet datastore

	sqliteDatastore, _ := db.Create("/home/dev/gex/go-electrum-client/wallet/btc/testdata")
	// sqliteDatastore, _ := db.Create(cfg.DataDir)
	cfg.DB = sqliteDatastore

	return &BtcElectrumClient{
		config:       cfg,
		chainManager: manager,
	}
}

func newchainManager(cfg *wallet.Config) *wallet.ChainManager {
	manager := wallet.NewChainManager(cfg)
	return manager
}

func (ec *BtcElectrumClient) CreateWallet(privPass string) error {

	ec.wallet = BtcElectrumWallet{
		Asset: "BTC",
		Net:   ec.config.Params,
	}

	return nil
}

/*
ElectrumWallet
*/
var _ = (*wallet.ElectrumWallet)(nil)

// BtcElectrumWallet implements ElectrumWallet
type BtcElectrumWallet struct {
	Asset string
	Net   *chaincfg.Params
}
