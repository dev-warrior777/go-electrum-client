package btc

import (
	"context"
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
	// cancel stale addressStatusNotify thread after network restart
	cancelAddressStatusNotify context.CancelFunc
	// cancel stale headersNotify thread after network restart
	cancelHeadersNotify context.CancelFunc
}

func NewBtcElectrumClient(cfg *client.ClientConfig) client.ElectrumClient {
	ec := BtcElectrumClient{
		ClientConfig:              cfg,
		Wallet:                    nil,
		Node:                      nil,
		cancelAddressStatusNotify: nil,
		cancelHeadersNotify:       nil,
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

// createNode creates a single unconnected ElectrumX node
func (ec *BtcElectrumClient) createNode(_ client.NodeType) error {
	nodeCfg := ec.GetConfig().MakeNodeConfig()
	n, err := elxbtc.NewSingleNode(nodeCfg)
	if err != nil {
		return err
	}
	ec.Node = n
	return nil
}

// client interface implementation

func (ec *BtcElectrumClient) Start(ctx context.Context) error {
	err := ec.createNode(client.SingleNode)
	if err != nil {
		return err
	}
	err = ec.Node.Start(ctx)
	if err != nil {
		return err
	}
	err = ec.syncHeaders(ctx)
	if err != nil {
		return err
	}
	err = ec.listenNetworkRestarted(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (ec *BtcElectrumClient) listenNetworkRestarted(ctx context.Context) error {
	node := ec.GetNode()
	if node == nil {
		return ErrNoNode
	}
	networkRestartCh := node.RegisterNetworkRestart()
	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("listenNetworkRestarted thread exit")
				return
			case nr := <-networkRestartCh:
				if nr == nil {
					// fmt.Println("network restart nr == <nil>")
					continue
				}
				fmt.Printf("network restart at %v\n", nr.Time)
				if ec.cancelHeadersNotify != nil {
					ec.cancelHeadersNotify()
					ec.cancelHeadersNotify = nil
				} else {
					fmt.Println("network restart cancelHeadersNotify == <nil>")
				}
				ec.syncHeaders(ctx)
				w := ec.GetWallet()
				if w != nil {
					if ec.cancelAddressStatusNotify != nil {
						ec.cancelAddressStatusNotify()
						ec.cancelAddressStatusNotify = nil
					} else {
						fmt.Println("network restart cancelAddressStatusNotify == <nil>")
					}
					ec.SyncWallet(ctx)
				}
			}
		}
	}()
	return nil
}

func (ec *BtcElectrumClient) Stop() {
	fmt.Printf("client stopping\n")
	ec.CloseWallet()
	node := ec.GetNode()
	if node != nil {
		node.Stop()
	}
	fmt.Printf("client stopped\n")
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
func (ec *BtcElectrumClient) RecreateWallet(ctx context.Context, pw, mnenomic string) error {
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
	err = ec.RescanWallet(ctx)
	if err != nil {
		return err
	}
	return nil
}

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
// GetBlockHeader(height int64) *wire.BlockHeader
// GetBlockHeaders(startHeight, count int64) ([]*wire.BlockHeader, error)
// RegisterTipChangeNotify(tipChange func(height int64)) error
// UnegisterTipChangeNotify()

// Interface methods in client_wallet.go
//
// Spend(amount int64, toAddress string, feeLevel wallet.FeeLevel, broadcast bool) (string, string, error)
// GetPrivKeyForAddress(pw, addr string) (string, error)
// Broadcast(*BroadcastParams) (string, error)
// ListUnspent() ([]wallet.Utxo, error)
// UnusedAddress() (string, error)
// ChangeAddress() (string, error)
// Balance() (int64, int64, error)
// FreezeUTXO((txid string, out uint32) error
// UnFreezeUTXO((txid string, out uint32) error

// Interface methods in client_node.go
//
// GetTransaction(txid string) (*electrumx.GetTransactionResult, error)
// GetRawTransaction(txid string) ([]byte, error)
// GetAddressHistory(addr string) (electrumx.HistoryResult, error)
// GetAddressUnspent(addr string) (electrumx.ListUnspentResult, error)
//
//////////////////////////////////////////////////////////////////////////////
