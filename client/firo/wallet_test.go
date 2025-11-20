// This code is available on the terms of the project LICENSE.md file,
// also available online at https://blueoakcouncil.org/license/1.0.0.

package firo

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bisoncraft/go-electrum-client/client"
	"github.com/bisoncraft/go-electrum-client/wallet"
	"github.com/btcsuite/btcd/chaincfg"
)

func makeBitcoinRegtestTestConfig(dataDir string) (*client.ClientConfig, error) {
	cfg := client.NewDefaultConfig()
	cfg.CoinType = wallet.Bitcoin
	cfg.DataDir = dataDir
	cfg.Params = &chaincfg.RegressionNetParams
	cfg.StoreEncSeed = true
	regtestTestDir := filepath.Join(dataDir, "btc", "regtest", "test")
	err := os.MkdirAll(regtestTestDir, os.ModeDir|0777)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}


// Create a new standard wallet
func TestWalletCreation(t *testing.T) {
	dataDir := t.TempDir()
	cfg, err := makeBitcoinRegtestTestConfig(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Testing = true
	ec := NewFiroElectrumClient(cfg)
	pw := "abc"
	err = ec.CreateWallet(pw)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("made a btcWallet")

	adr, err := ec.GetWallet().GetUnusedAddress(wallet.EXTERNAL)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Current External address", adr)
	adrI, err := ec.GetWallet().GetUnusedAddress(wallet.INTERNAL)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("Current Internal address", adrI)
}
