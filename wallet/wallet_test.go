package wallet

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestWalletCreationAndLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	walletFile := filepath.Join(tmpDir, "wallet.db")
	fmt.Printf("%s\n", walletFile)
	//privPass := "abc"

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
