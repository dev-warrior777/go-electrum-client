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

func TestWalletCreationAndLoad(t *testing.T) {
	cfg := wallet.NewDefaultConfig()
	cfg.Chain = wallet.Bitcoin
	cfg.Params = &chaincfg.TestNet3Params
	cfg.DataDir = tmpDirPath
	walletFile := filepath.Join(cfg.DataDir, "wallet.db")
	fmt.Printf("Wallet: %s\n", walletFile)

	ec := NewBtcElectrumClient(cfg)
	fmt.Println("ChainManager: ", ec.chainManager)

	privPass := "abc"
	err := ec.CreateWallet(privPass)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("made a btcWallet", ec.wallet)

	adr := ec.wallet.CurrentAddress(wallet.EXTERNAL)
	fmt.Println("Current External address", adr)
	newAdr := ec.wallet.NewAddress(wallet.EXTERNAL)
	fmt.Println("New External address", newAdr)
	newAdr1 := ec.wallet.NewAddress(wallet.EXTERNAL)
	fmt.Println("New External address", newAdr1)
	newAdr2 := ec.wallet.NewAddress(wallet.EXTERNAL)
	fmt.Println("New External address", newAdr2)
	adr2 := ec.wallet.CurrentAddress(wallet.EXTERNAL)
	fmt.Println("Current External address 2", adr2)

	adrI := ec.wallet.CurrentAddress(wallet.INTERNAL)
	fmt.Println("Current Internal address", adrI)
	adrNewI := ec.wallet.NewAddress(wallet.INTERNAL)
	fmt.Println("New Internal address", adrNewI)
	adrI2 := ec.wallet.CurrentAddress(wallet.INTERNAL)
	fmt.Println("Current Internal address 2", adrI2)
	/*



	  Some things a wallet can do



	*/
	// seed, err := hdkeychain.GenerateSeed(hdkeychain.RecommendedSeedLen)
	// if err != nil {
	//	t.Fatal(err)
	//}
	// wallet, err := Create(file, privPass, seed)
	// if err != nil {
	//	t.Fatal(err)
	//}
	// fmt.Printf("%v\n", wallet)

	// if addrs, err := wallet.Addresses(); err != nil {
	// 	t.Fatal(err)
	// } else if len(addrs) != 0 {
	// 	t.Fatalf("wallet doesn't start with 0 addresses, len = %d", len(addrs))
	// }

	// if addrs, err := wallet.GenAddresses(10); err != nil {
	// 	t.Fatal(err)
	// } else if len(addrs) != 10 {
	// 	t.Fatalf("generated wrong number of addresses, len = %d", len(addrs))
	// }

	// if addrs, err := wallet.Addresses(); err != nil {
	// 	t.Fatal(err)
	// } else if len(addrs) != 10 {
	// 	t.Fatalf("wallet doesn't have new addresses, len = %d", len(addrs))
	// } else {
	// 	for _, addr := range addrs {
	// 		fmt.Printf("addr %s\n", addr.String())
	// 	}
	// }
	// err = wallet.SendBitcoin(map[string]cashutil.Amount{"171RiZZqGzgB25Wxn3MKqo4JsjkMNSJFJe": 0}, 0)
	// if err != nil {
	// 	t.Fatal(err)
	// }
}
