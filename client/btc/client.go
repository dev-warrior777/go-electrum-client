package btc

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/dev-warrior777/go-electrum-client/client"
	"github.com/dev-warrior777/go-electrum-client/electrumx"
	"github.com/dev-warrior777/go-electrum-client/wallet"
	"github.com/dev-warrior777/go-electrum-client/wallet/db"
	"github.com/dev-warrior777/go-electrum-client/wallet/wltbtc"
)

// BtcElectrumClient
type BtcElectrumClient struct {
	ClientConfig *client.ClientConfig
	Wallet       wallet.ElectrumWallet
	Node         electrumx.ElectrumXNode
}

func NewBtcElectrumClient(cfg *client.ClientConfig) client.ElectrumClient {
	ec := BtcElectrumClient{
		ClientConfig: cfg,
		// Wallet: nil,
		// Node:   nil,
	}
	return &ec
}

func (ec *BtcElectrumClient) MakeWalletConfig() *wallet.WalletConfig {
	wc := wallet.WalletConfig{
		Chain:        ec.ClientConfig.Chain,
		Params:       ec.ClientConfig.Params,
		StoreEncSeed: ec.ClientConfig.StoreEncSeed,
		DataDir:      ec.ClientConfig.DataDir,
		DB:           ec.ClientConfig.DB,
		LowFee:       ec.ClientConfig.LowFee,
		MediumFee:    ec.ClientConfig.MediumFee,
		HighFee:      ec.ClientConfig.HighFee,
		MaxFee:       ec.ClientConfig.MaxFee,
		Testing:      ec.ClientConfig.Testing,
	}
	return &wc
}

func (ec *BtcElectrumClient) MakeNodeConfig() *electrumx.NodeConfig {
	nc := electrumx.NodeConfig{
		Chain:       ec.ClientConfig.Chain,
		Params:      ec.ClientConfig.Params,
		UserAgent:   ec.ClientConfig.UserAgent,
		DataDir:     ec.ClientConfig.DataDir,
		TrustedPeer: ec.ClientConfig.TrustedPeer,
		Proxy:       ec.ClientConfig.Proxy,
		Testing:     ec.ClientConfig.Testing,
	}
	return &nc
}

func (ec *BtcElectrumClient) GetConfig() *client.ClientConfig {
	return ec.ClientConfig
}

func (ec *BtcElectrumClient) GetWallet() wallet.ElectrumWallet {
	return ec.Wallet
}

func (ec *BtcElectrumClient) GetNode() electrumx.ElectrumXNode {
	return ec.Node
}

// CreateWallet makes a new wallet with a new seed. The password is to encrypt
// stored xpub, xprv and other sensitive data.
func (ec *BtcElectrumClient) CreateWallet(pw string) error {
	cfg := ec.ClientConfig
	datadir := ec.ClientConfig.DataDir
	if _, err := os.Stat(path.Join(datadir, "wallet.db")); err == nil {
		if !ec.ClientConfig.Testing {
			return errors.New("wallet.db already exists")
		}
		fmt.Printf("a file wallet.db probably exists in the datadir: %s .. \n"+
			"test will overwrite\n", cfg.DataDir)
	}

	// Select wallet datastore
	sqliteDatastore, err := db.Create(cfg.DataDir)
	if err != nil {
		return err
	}
	cfg.DB = sqliteDatastore

	walletCfg := ec.MakeWalletConfig()
	ec.Wallet, err = wltbtc.NewBtcElectrumWallet(walletCfg, pw)
	if err != nil {
		return err
	}

	return nil
}

// func NewBtcElectrumWallet(cfg *client.Config, pw string) {
// 	panic("unimplemented")
// }

// RecreateElectrumWallet recreates a wallet from an existing mnemonic seed.
// The password is to encrypt the stored xpub, xprv and other sensitive data
// and can be different from the original wallet's password.
func (ec *BtcElectrumClient) RecreateElectrumWallet(pw, mnenomic string) error {
	cfg := ec.ClientConfig
	datadir := ec.ClientConfig.DataDir
	if _, err := os.Stat(path.Join(datadir, "wallet.db")); err == nil {
		if !ec.ClientConfig.Testing {
			return errors.New("wallet.db already exists")
		}
		fmt.Printf("a file wallet.db probably exists in the datadir: %s .. \n"+
			"test will overwrite\n", cfg.DataDir)
	}

	// Select wallet datastore
	sqliteDatastore, err := db.Create(cfg.DataDir)
	if err != nil {
		return err
	}
	cfg.DB = sqliteDatastore

	walletCfg := ec.MakeWalletConfig()
	ec.Wallet, err = wltbtc.RecreateElectrumWallet(walletCfg, pw, mnenomic)
	if err != nil {
		return err
	}

	return nil
}

// LoadWallet loads an existing wallet. The password is required to decrypt
// the stored xpub, xprv and other sensitive data
func (ec *BtcElectrumClient) LoadWallet(pw string) error {
	cfg := ec.ClientConfig

	// Select wallet datastore
	sqliteDatastore, err := db.Create(cfg.DataDir)
	if err != nil {
		return err
	}
	cfg.DB = sqliteDatastore

	walletCfg := ec.MakeWalletConfig()
	ec.Wallet, err = wltbtc.LoadBtcElectrumWallet(walletCfg, pw)
	if err != nil {
		return err
	}

	return nil
}

// CreateNode creates an ElectrumX node - single or multi
func (ec *BtcElectrumClient) CreateNode() error {
	nodeCfg := ec.MakeNodeConfig()
	ec.Node = electrumx.SingleNode{
		NodeConfig: nodeCfg,
		Server:     &electrumx.ServerConn{},
	}

	return nil
}
