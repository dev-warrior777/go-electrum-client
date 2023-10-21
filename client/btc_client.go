package client

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/dev-warrior777/go-electrum-client/electrumx"
	"github.com/dev-warrior777/go-electrum-client/wallet"
	"github.com/dev-warrior777/go-electrum-client/wallet/btc"
	"github.com/dev-warrior777/go-electrum-client/wallet/db"
)

// BtcElectrumClient
type BtcElectrumClient struct {
	config *Config
	wallet wallet.ElectrumWallet
}

func NewBtcElectrumClient(cfg *Config) ElectrumClient {
	return &BtcElectrumClient{
		config: cfg,
		wallet: nil,
	}
}

func (ec *BtcElectrumClient) MakeWalletConfig() *wallet.WalletConfig {
	wc := wallet.WalletConfig{
		Chain:        ec.config.Chain,
		Params:       ec.config.Params,
		StoreEncSeed: ec.config.StoreEncSeed,
		DataDir:      ec.config.DataDir,
		DB:           ec.config.DB,
		LowFee:       ec.config.LowFee,
		MediumFee:    ec.config.MediumFee,
		HighFee:      ec.config.HighFee,
		MaxFee:       ec.config.MaxFee,
		Testing:      ec.config.Testing,
	}
	return &wc
}

// CreateWallet makes a new wallet with a new seed. The password is to encrypt
// stored xpub, xprv and other sensitive data.
func (ec *BtcElectrumClient) CreateWallet(pw string) error {
	cfg := ec.config
	datadir := ec.config.DataDir
	if _, err := os.Stat(path.Join(datadir, "wallet.db")); err == nil {
		if !ec.config.Testing {
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
	ec.wallet, err = btc.NewBtcElectrumWallet(walletCfg, pw)
	if err != nil {
		return err
	}

	return nil
}

func NewBtcElectrumWallet(cfg *Config, pw string) {
	panic("unimplemented")
}

// RecreateElectrumWallet recreates a wallet from an existing mnemonic seed.
// The password is to encrypt the stored xpub, xprv and other sensitive data
// and can be different from the original wallet's password.
func (ec *BtcElectrumClient) RecreateElectrumWallet(pw, mnenomic string) error {
	cfg := ec.config
	datadir := ec.config.DataDir
	if _, err := os.Stat(path.Join(datadir, "wallet.db")); err == nil {
		if !ec.config.Testing {
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
	ec.wallet, err = btc.RecreateElectrumWallet(walletCfg, pw, mnenomic)
	if err != nil {
		return err
	}

	return nil
}

// LoadWallet loads an existing wallet. The password is required to decrypt
// the stored xpub, xprv and other sensitive data
func (ec *BtcElectrumClient) LoadWallet(pw string) error {
	cfg := ec.config

	// Select wallet datastore
	sqliteDatastore, err := db.Create(cfg.DataDir)
	if err != nil {
		return err
	}
	cfg.DB = sqliteDatastore

	walletCfg := ec.MakeWalletConfig()
	ec.wallet, err = btc.LoadBtcElectrumWallet(walletCfg, pw)
	if err != nil {
		return err
	}

	return nil
}

func (ec *BtcElectrumClient) Config() *Config {
	return ec.config
}

func (ec *BtcElectrumClient) Wallet() wallet.ElectrumWallet {
	return ec.wallet
}

func (ec *BtcElectrumClient) Node() electrumx.ElectrumXNode {
	return ec.Node
}
