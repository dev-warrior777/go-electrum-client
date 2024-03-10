package btc

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/dev-warrior777/go-electrum-client/client"
	"github.com/dev-warrior777/go-electrum-client/electrumx"
	"github.com/dev-warrior777/go-electrum-client/electrumx/elxbtc"
	"github.com/dev-warrior777/go-electrum-client/wallet"
	"github.com/dev-warrior777/go-electrum-client/wallet/bdb"
	"github.com/dev-warrior777/go-electrum-client/wallet/db"
	"github.com/dev-warrior777/go-electrum-client/wallet/wltbtc"
)

// BtcElectrumClient
type BtcElectrumClient struct {
	ClientConfig *client.ClientConfig
	Wallet       wallet.ElectrumWallet
	Node         electrumx.ElectrumXNode
	// client copy of blockchain headers
	clientHeaders *Headers
}

func NewBtcElectrumClient(cfg *client.ClientConfig) client.ElectrumClient {
	ec := BtcElectrumClient{
		ClientConfig: cfg,
		Wallet:       nil,
		Node:         nil,
	}
	ec.clientHeaders = NewHeaders(cfg)
	return &ec
}

//////////////////////////////////////////////////////////////////////////////
// Interface impl
/////////////////

func (ec *BtcElectrumClient) GetConfig() *client.ClientConfig {
	return ec.ClientConfig
}

func (ec *BtcElectrumClient) GetWallet() wallet.ElectrumWallet {
	return ec.Wallet
}

func (ec *BtcElectrumClient) GetNode() electrumx.ElectrumXNode {
	return ec.Node
}

func (ec *BtcElectrumClient) walletExists() bool {
	cfg := ec.ClientConfig
	datadir := ec.ClientConfig.DataDir
	var walletName = ""
	switch cfg.DbType {
	case client.DbTypeBolt:
		walletName = "wallet.bdb"
	case client.DbTypeSqlite:
		walletName = "wallet.db"
	}
	if _, err := os.Stat(path.Join(datadir, walletName)); err != nil {
		return false
	}
	return true
}

func (ec *BtcElectrumClient) getDatastore() error {
	cfg := ec.ClientConfig
	switch cfg.DbType {
	case client.DbTypeBolt:
		// Select a bbolt wallet datastore - false = RW database
		boltDatastore, err := bdb.Create(cfg.DataDir, false)
		if err != nil {
			return err
		}
		cfg.DB = boltDatastore
	case client.DbTypeSqlite:
		// Select a sqlite wallet datastore
		sqliteDatastore, err := db.Create(cfg.DataDir)
		if err != nil {
			return err
		}
		cfg.DB = sqliteDatastore
	default:
		return errors.New("unknown database type")
	}
	return nil
}

func (ec *BtcElectrumClient) Start() error {
	ec.createNode(client.SingleNode)
	err := ec.Node.Start()
	if err != nil {
		return err
	}
	return ec.syncHeaders()
}

func (ec *BtcElectrumClient) Stop() {
	fmt.Printf("client.Stopping\n")
	ec.CloseWallet()
	node := ec.GetNode()
	if node != nil {
		node.Stop()
	}
	fmt.Printf("client.Stop\n")
}

// CreateWallet makes a new wallet with a new seed. The password is to encrypt
// stored xpub, xprv and other sensitive data.
func (ec *BtcElectrumClient) CreateWallet(pw string) error {
	if ec.walletExists() {
		return errors.New("wallet already exists")
	}
	err := ec.getDatastore()
	if err != nil {
		return err
	}

	walletCfg := ec.ClientConfig.MakeWalletConfig()

	ec.Wallet, err = wltbtc.NewBtcElectrumWallet(walletCfg, pw)
	if err != nil {
		return err
	}
	return nil
}

// RecreateWallet recreates a wallet from an existing mnemonic seed.
// The password is to encrypt the stored xpub, xprv and other sensitive data
// and can be different from the original wallet's password.
func (ec *BtcElectrumClient) RecreateWallet(pw, mnenomic string) error {
	if ec.walletExists() {
		//TODO: should we backup any wallet file that exists
		return errors.New("wallet already exists")
	}
	err := ec.getDatastore()
	if err != nil {
		return err
	}
	walletCfg := ec.ClientConfig.MakeWalletConfig()
	ec.Wallet, err = wltbtc.RecreateElectrumWallet(walletCfg, pw, mnenomic)
	if err != nil {
		return err
	}
	// Do a rescan as alhough we have a wallet structure with a keychain we
	// do not have any transaction history
	err = ec.RescanWallet()
	if err != nil {
		return err
	}
	return nil
}

// createNode creates a single unconnected ElectrumX node
func (ec *BtcElectrumClient) createNode(_ client.NodeType) {
	nodeCfg := ec.GetConfig().MakeNodeConfig()
	n := elxbtc.NewSingleNode(nodeCfg)
	ec.Node = n
}

// client interface implementation

// LoadWallet loads an existing wallet. The password is required to decrypt
// the stored xpub, xprv and other sensitive data
func (ec *BtcElectrumClient) LoadWallet(pw string) error {
	if !ec.walletExists() {
		return errors.New("cannot find wallet")
	}
	err := ec.getDatastore()
	if err != nil {
		return err
	}
	walletCfg := ec.ClientConfig.MakeWalletConfig()
	ec.Wallet, err = wltbtc.LoadBtcElectrumWallet(walletCfg, pw)
	if err != nil {
		return err
	}
	return nil
}

func (ec *BtcElectrumClient) CloseWallet() {
	w := ec.GetWallet()
	if w != nil {
		w.Close()
	}
}

// Interface methods in client_headers.go
//
// Tip() (int64, bool)

// Interface methods in client_wallet.go
//
// Spend(amount int64, toAddress string, feeLevel wallet.FeeLevel, broadcast bool) (string, string, error)
// Broadcast(*BroadcastParams) (string, error)
// ListUnspent() (string, error)
// UnusedAddress() (string, error)
// ChangeAddress() (string, error)
// Balance() (int64, int64, error)
// FreezeUTXO((txid string, out uint32) error
// UnFreezeUTXO((txid string, out uint32) error

//////////////////////////////////////////////////////////////////////////////
