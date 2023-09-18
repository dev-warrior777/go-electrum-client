package btc

import (
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/dev-warrior777/go-electrum-client/chain"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

// BtcElectrumClient
type BtcElectrumClient struct {
	wallet       wallet.ElectrumWallet
	chainManager *chain.ChainManager
}

func NewBtcElectrumClient(net chain.NetType) *BtcElectrumClient {
	manager := chain.NewChainManager(net, "BTC")
	return &BtcElectrumClient{
		chainManager: manager,
	}
}

func (ec *BtcElectrumClient) CreateWallet(privPass string) error {
	var chainParams chaincfg.Params
	switch ec.chainManager.Net {
	case "mainnet":
		chainParams = chaincfg.MainNetParams
	case "testnet":
		chainParams = chaincfg.TestNet3Params
	default:
		return fmt.Errorf("invalid network %s", ec.chainManager.Net)
	}

	ec.wallet = BtcElectrumWallet{
		Asset: "BTC",
		Net:   chainParams,
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
	Net   chaincfg.Params
}
