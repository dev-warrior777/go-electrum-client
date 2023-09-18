package btc

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"testing"
	"time"

	"github.com/dev-warrior777/go-electrum-client/chain"
	chainbtc "github.com/dev-warrior777/go-electrum-client/chain/btc"
)

func TestWalletCreationAndLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatalf("Error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	walletFile := filepath.Join(tmpDir, "wallet.db")
	fmt.Printf("%s\n", walletFile)

	ec := NewBtcElectrumClient("testnet")
	fmt.Println("ChainManager: ", ec.chainManager)

	privPass := "abc"
	err = ec.CreateWallet(privPass)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("made a btcWallet", ec.wallet)

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

var (
	// Should specify a available server(IP:PORT) if connecting to the
	// following server failed.

	// https://github.com/spesmilo/electrum/blob/cffbe44c07a59a7d6a3d5183181659a57de8d2c0/electrum/servers_testnet.json
	testServerAddr = "blockstream.info:993"

	// https://github.com/spesmilo/electrum/blob/cffbe44c07a59a7d6a3d5183181659a57de8d2c0/electrum/servers.json
	mainserverAddr = "blockstream.info:700"
)

func TestRunNode(t *testing.T) {
	ec := NewBtcElectrumClient("testnet")
	// ec := NewBtcElectrumClient("mainnet")
	cm := ec.chainManager
	fmt.Println("NodeManager: ", cm)

	var serverAddr string
	switch cm.Net {
	case chain.NetTypeMainnet:
		serverAddr = mainserverAddr
	case chain.NetTypeTestnet:
		serverAddr = testServerAddr
	default:
		t.Fatal("invalid chain net type")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	chainbtc.DebugMode = true

	btcNode := chainbtc.NewNode()
	defer btcNode.Disconnect()

	connectCtx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	if err := btcNode.Connect(connectCtx, serverAddr, &tls.Config{}); err != nil {
		e := fmt.Errorf("connect node: %w", err).Error()
		t.Fatalf("connect node %s", e)
	}

	// Remember to drain errors! Since communication is async not all errors
	// will happen as a direct response to requests.
	go func() {
		err := <-btcNode.Errors()
		log.Printf("ran into error: %s", err)
		btcNode.Disconnect()
	}()

	//Send server.ping request in order to keep alive connection to
	//electrum server
	go func() {
		for {
			if err := btcNode.Ping(ctx); err != nil {
				log.Fatal(err)
			}

			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return
			}

		}
	}()

	version, err := btcNode.ServerVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Version: %v\n\n", version)

	banner, err := btcNode.ServerBanner(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Banner: %s\n\n", banner)

	address, err := btcNode.ServerDonationAddress(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Address: %s\n\n", address)

	peers, err := btcNode.ServerPeersSubscribe(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Peers: %+v\n\n", peers)

	headerChan, err := btcNode.BlockchainHeadersSubscribe(ctx)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for header := range headerChan {
			fmt.Printf("Headers: %+v\n\n", header)
		}
	}()

	var transaction *chainbtc.GetTransaction
	switch cm.Net {
	case chain.NetTypeMainnet:
		transaction, err = btcNode.BlockchainTransactionGet( /*mainnet*/
			ctx, "f53a8b83f85dd1ce2a6ef4593e67169b90aaeb402b3cf806b37afc634ef71fbc", false)
	case chain.NetTypeTestnet:
		transaction, err = btcNode.BlockchainTransactionGet( /*testnet3*/
			ctx, "581d837b8bcca854406dc5259d1fb1e0d314fcd450fb2d4654e78c48120e0135", false)
	default:
		t.Fatal("invalid chain net type")
	}
	if err != nil {
		fmt.Printf("blockchain transaction get: %s\n", err)
	} else {
		fmt.Printf("Transaction: %+v\n\n", transaction.Hex)
	}

	// If you're connecting to a node that support address queries, change this
	// to true.
	nodeSupportsAddressQueries := false
	if nodeSupportsAddressQueries {
		balance, err := btcNode.BlockchainAddressGetBalance(ctx, "bc1qv5wppm0xwarzwea9xxascc5rne7c0c296h7y5p")
		if err != nil {
			t.Fatal(err)
		}

		fmt.Printf("address balance: %+v\n\n", balance)
	}

	const until = time.Second * 7
	fmt.Printf("leaving connection open for %s\n", until)
	select {
	case <-ctx.Done():
		t.Fatal(ctx.Err())

	case <-time.After(until):
	}
}
