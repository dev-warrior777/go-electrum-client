package client

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

// new wallets, header files, etc. Manually clean while developing
const coinDir = "btc"

func makeBitcoinRegtestConfig() (*Config, error) {
	cfg := NewDefaultConfig()
	cfg.Chain = wallet.Bitcoin
	cfg.Params = &chaincfg.RegressionNetParams
	cfg.StoreEncSeed = true
	appDir, err := GetConfigPath()
	if err != nil {
		return nil, err
	}
	regtestDir := filepath.Join(appDir, coinDir, "regtest")
	err = os.MkdirAll(regtestDir, os.ModeDir|0777)
	if err != nil {
		return nil, err
	}
	cfg.DataDir = regtestDir
	return cfg, nil
}

// Create a new standard wallet
func TestWalletCreation(t *testing.T) {
	cfg, err := makeBitcoinRegtestConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Testing = true
	ec := NewBtcElectrumClient(cfg)

	pw := "abc"
	err = ec.CreateWallet(pw)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("made a btcWallet", ec.Wallet())

	adr := ec.Wallet().CurrentAddress(wallet.EXTERNAL)
	fmt.Println("Current External address", adr)
	adrI := ec.Wallet().CurrentAddress(wallet.INTERNAL)
	fmt.Println("Current Internal address", adrI)
}

var mnemonic = "jungle pair grass super coral bubble tomato sheriff pulp cancel luggage wagon"

// var seedForMnenomic = "148e047034a3f0a88905f9c2fa08bce280681db23d1f38783d3980a6cfbe327439159a51068343c274dc8819bd150fa018faffbe76133989f936a21e6b7bd0ed"

// Recreate a known wallet. Overwrites 'wallet.db' of the previous test
func TestWalletRecreate(t *testing.T) {
	cfg, err := makeBitcoinRegtestConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Testing = true
	ec := NewBtcElectrumClient(cfg)
	pw := "abc"
	err = ec.RecreateElectrumWallet(pw, mnemonic)
	if err != nil {
		t.Fatal(err)
	}
}

// Load the recreated wallet with known seed
func TestWalletLoad(t *testing.T) {
	cfg, err := makeBitcoinRegtestConfig()
	cfg.Testing = true
	if err != nil {
		t.Fatal(err)
	}
	pw := "abc"
	ec := NewBtcElectrumClient(cfg)
	err = ec.LoadWallet(pw)
	if err != nil {
		t.Fatal(err)
	}
}
