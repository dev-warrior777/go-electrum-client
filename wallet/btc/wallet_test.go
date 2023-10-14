package btc

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

// new wallets, etc. Manually clean while developing
const tmpDirName = "testdata"

var tmpDirPath string

func init() {
	wd, _ := os.Getwd()
	tmpDirPath = filepath.Join(wd, tmpDirName)
}
func makeBitcoinTestnetConfig() *wallet.Config {
	cfg := wallet.NewDefaultConfig()
	cfg.Chain = wallet.Bitcoin
	cfg.Params = &chaincfg.TestNet3Params
	cfg.StoreEncSeed = true
	cfg.DataDir = tmpDirPath
	return cfg
}

// Create a new standard wallet
func TestWalletCreation(t *testing.T) {
	cfg := makeBitcoinTestnetConfig()
	ec := NewBtcElectrumClient(cfg)

	pw := "abc"
	err := ec.CreateWallet(pw)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("made a btcWallet", ec.wallet)

	adr := ec.wallet.CurrentAddress(wallet.EXTERNAL)
	fmt.Println("Current External address", adr)
	adrI := ec.wallet.CurrentAddress(wallet.INTERNAL)
	fmt.Println("Current Internal address", adrI)
}

var mnemonic = "jungle pair grass super coral bubble tomato sheriff pulp cancel luggage wagon"

// var seedForMnenomic = "148e047034a3f0a88905f9c2fa08bce280681db23d1f38783d3980a6cfbe327439159a51068343c274dc8819bd150fa018faffbe76133989f936a21e6b7bd0ed"

// Recreate a known wallet. Overwrites 'wallet.db' of the previous test
func TestWalletRecreate(t *testing.T) {
	cfg := makeBitcoinTestnetConfig()
	ec := NewBtcElectrumClient(cfg)
	pw := "abc"
	err := ec.RecreateElectrumWallet(pw, mnemonic)
	if err != nil {
		t.Fatal(err)
	}
}

// Load the recreated wallet
func TestWalletLoad(t *testing.T) {
	cfg := makeBitcoinTestnetConfig()
	pw := "abc"
	ec := NewBtcElectrumClient(cfg)
	err := ec.LoadWallet(pw)
	if err != nil {
		t.Fatal(err)
	}
}
