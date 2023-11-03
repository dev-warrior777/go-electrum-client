package btc

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/dev-warrior777/go-electrum-client/client"
	"github.com/dev-warrior777/go-electrum-client/electrumx"
	"github.com/dev-warrior777/go-electrum-client/electrumx/elxbtc"
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
		Wallet:       nil,
		Node:         nil,
	}
	return &ec
}

//////////////////////////////////////////////////////////////////////////////
// Interface
////////////

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

	walletCfg := cfg.MakeWalletConfig()
	ec.Wallet, err = wltbtc.NewBtcElectrumWallet(walletCfg, pw)
	if err != nil {
		return err
	}
	return nil
}

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

	walletCfg := cfg.MakeWalletConfig()
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

	walletCfg := cfg.MakeWalletConfig()
	ec.Wallet, err = wltbtc.LoadBtcElectrumWallet(walletCfg, pw)
	if err != nil {
		return err
	}
	return nil
}

// CreateNode creates a single unconnected ElectrumX node
func (ec *BtcElectrumClient) CreateNode() {
	nodeCfg := ec.GetConfig().MakeNodeConfig()
	n := elxbtc.NewSingleNode(nodeCfg)
	ec.Node = n
}

// SyncHeaders reads blockchain_headers file, then gets any missing block from
// end of file to current tip from server. The current set of headers is also
// stored in headers map and the chain veirfied by checking previous block
// hashes backwards from Tip.
func (ec *BtcElectrumClient) SyncHeaders() error {
	headers, err := NewHeaders(ec.ClientConfig)
	if err != nil {
		return err
	}

	// 1. Read last stored blockchain_headers file for this network

	b, err := headers.ReadAllBytesFromFile()
	if err != nil {
		return err
	}
	lb := len(b)
	fmt.Println("read header bytes", lb)
	numHeaders, err := headers.BytesToNumHdrs(lb)
	if err != nil {
		return err
	}
	b = nil // gc

	maybeTip := numHeaders - 1

	// 2. Gather new block headers we did not have in file up to current tip

	// Do not make block count too big or electrumX may throttle response
	// as an anti ddos measure. Magic number 2016 from electrum code
	const blockDelta = 20 // 20 dev 2016 pro
	var doneGathering = false
	var startHeight = uint32(numHeaders)
	var blockCount = uint32(20)

	n := ec.GetNode()

	hdrsRes, err := n.BlockHeaders(startHeight, blockCount)
	if err != nil {
		return err
	}
	count := hdrsRes.Count

	fmt.Print("Count: ", count, " read from server at Height: ", startHeight)

	if count > 0 {
		b, err := hex.DecodeString(hdrsRes.HexConcat)
		if err != nil {
			log.Fatal(err)
		}
		nh, err := headers.AppendHeaders(b)
		if err != nil {
			log.Fatal(err)
		}
		maybeTip += int32(count)

		fmt.Println(" Appended: ", nh, " headers at ", startHeight, " maybeTip ", maybeTip)
	}

	if count < blockDelta {
		fmt.Println("\nDone gathering")
		doneGathering = true
	}

	sc, err := n.GetServerConn()
	if err != nil {
		return err
	}
	svrCtx := sc.SvrCtx

	for !doneGathering {

		startHeight += blockDelta

		select {
		case <-svrCtx.Done():
			fmt.Println("Server shutdown - gathering")
			n.Stop()
			return nil
		case <-time.After(time.Millisecond * 33):
			hdrsRes, err := n.BlockHeaders(startHeight, blockCount)
			if err != nil {
				return err
			}
			count = hdrsRes.Count

			fmt.Print("Count: ", count, " read from server at Height: ", startHeight)

			if count > 0 {
				b, err := hex.DecodeString(hdrsRes.HexConcat)
				if err != nil {
					return err
				}
				nh, err := headers.AppendHeaders(b)
				if err != nil {
					return err
				}
				maybeTip += int32(count)

				fmt.Println(" Appended: ", nh, " headers at ", startHeight, " maybeTip ", maybeTip)
			}

			if count < blockDelta {
				fmt.Println("\nDone gathering")
				doneGathering = true
			}
		}
	}

	// 3. Read up to date blockchain_headers file - this can be improved since
	//    we already read most of it but for now; simplicity
	b2, err := headers.ReadAllBytesFromFile()
	if err != nil {
		return err
	}

	// 4. Store all headers in a map
	err = headers.Store(b2, 0)
	if err != nil {
		return err
	}
	headers.hdrsTip = maybeTip

	// 5. Verify headers in headers map
	fmt.Printf("starting verify at height %d\n", headers.hdrsTip)
	err = headers.VerifyAll()
	if err != nil {
		return err
	}
	fmt.Println("header chain verified")

	headers.synced = true
	fmt.Println("headers synced up to tip ", headers.hdrsTip)
	return nil
}

//////////////////////////////////////////////////////////////////////////////
// Btc
//////
// func (ec *BtcElectrumClient) Foo() error {
