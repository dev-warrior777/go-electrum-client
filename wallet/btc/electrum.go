package btc

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/dev-warrior777/go-electrum-client/client"
	"github.com/dev-warrior777/go-electrum-client/wallet"
	"github.com/dev-warrior777/go-electrum-client/wallet/db"
)

// BtcElectrumClient
type BtcElectrumClient struct {
	config *client.Config
	wallet wallet.ElectrumWallet
}

func NewBtcElectrumClient(cfg *client.Config) *BtcElectrumClient {
	return &BtcElectrumClient{
		config: cfg,
	}
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

	ec.wallet, err = NewBtcElectrumWallet(cfg, pw)
	if err != nil {
		return err
	}

	return nil
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

	ec.wallet, err = RecreateElectrumWallet(cfg, pw, mnenomic)
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

	ec.wallet, err = LoadBtcElectrumWallet(cfg, pw)
	if err != nil {
		return err
	}

	return nil
}
