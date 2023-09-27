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
	cfg.DataDir = tmpDirPath
	return cfg
}
func TestWalletCreation(t *testing.T) {
	cfg := makeBitcoinTestnetConfig()

	ec := NewBtcElectrumClient(cfg)

	privPass := "abc"
	err := ec.CreateWallet(privPass)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("made a btcWallet", ec.wallet)

	adr := ec.wallet.CurrentAddress(wallet.EXTERNAL)
	fmt.Println("Current External address", adr)
	// newAdr := ec.wallet.NewAddress(wallet.EXTERNAL)
	// fmt.Println("New External address", newAdr)
	// newAdr1 := ec.wallet.NewAddress(wallet.EXTERNAL)
	// fmt.Println("New External address", newAdr1)
	// newAdr2 := ec.wallet.NewAddress(wallet.EXTERNAL)
	// fmt.Println("New External address", newAdr2)
	// adr2 := ec.wallet.CurrentAddress(wallet.EXTERNAL)
	// fmt.Println("Current External address 2", adr2)

	adrI := ec.wallet.CurrentAddress(wallet.INTERNAL)
	fmt.Println("Current Internal address", adrI)
	// adrNewI := ec.wallet.NewAddress(wallet.INTERNAL)
	// fmt.Println("New Internal address", adrNewI)
	// adrI2 := ec.wallet.CurrentAddress(wallet.INTERNAL)
	// fmt.Println("Current Internal address 2", adrI2)
}
func TestWalletLoad(t *testing.T) {
	cfg := makeBitcoinTestnetConfig()
	privPass := "abc"
	ec := NewBtcElectrumClient(cfg)
	err := ec.LoadWallet(privPass)
	if err != nil {
		t.Fatal(err)
	}
}
