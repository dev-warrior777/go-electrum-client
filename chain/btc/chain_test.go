package btc

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"os/signal"
	"testing"
	"time"

	"github.com/dev-warrior777/go-electrum-client/chain"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

var (
	// Should specify a available server(IP:PORT) if connecting to the
	// following server failed.

	// https://github.com/spesmilo/electrum/blob/cffbe44c07a59a7d6a3d5183181659a57de8d2c0/electrum/servers_testnet.json
	testServerAddr = "blockstream.info:993"

	// https://github.com/spesmilo/electrum/blob/cffbe44c07a59a7d6a3d5183181659a57de8d2c0/electrum/servers.json
	mainServerAddr = "blockstream.info:700"
)

func TestRunMainnetNode(t *testing.T) {
	cm := wallet.ChainManager{
		Chain: wallet.Bitcoin,
		Net:   "mainnet",
	}
	fmt.Println("ChainManager: ", cm)
	RunNode(t, cm.Net, mainServerAddr)
}

func TestRunTestnetNode(t *testing.T) {
	cm := wallet.ChainManager{
		Chain: wallet.Bitcoin,
		Net:   "testnet",
	}
	fmt.Println("ChainManager: ", cm)
	RunNode(t, cm.Net, testServerAddr)
}

func RunNode(t *testing.T, net, addr string) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	chain.DebugMode = true

	btcNode := NewBtcNode(net)
	defer btcNode.Disconnect()

	connectCtx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	if err := btcNode.Connect(connectCtx, addr, &tls.Config{}); err != nil {
		e := fmt.Errorf("connect node: %w", err).Error()
		t.Fatalf("connect node %s", e)
	}

	// Remember to drain errors! Since communication is async not all errors
	// will happen as a direct response to requests.
	// I do not like this but cannot think of a better way right now ...
	go func() {
		err := <-btcNode.Node.Errors()
		log.Printf("ran into error: %s", err)
		btcNode.Disconnect()
	}()

	// Start services: 5s server.ping requests for connection keep alive
	btcNode.Start()

	version, err := btcNode.Node.ServerVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Version: %v\n\n", version)

	banner, err := btcNode.Node.ServerBanner(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Banner: %s\n\n", banner)

	address, err := btcNode.Node.ServerDonationAddress(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Address: %s\n\n", address)

	/*
		Next: server.features
	*/

	peers, err := btcNode.Node.ServerPeersSubscribe(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("Peers: %+v\n\n", peers)

	headerChan, err := btcNode.Node.BlockchainHeadersSubscribe(ctx)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for header := range headerChan {
			fmt.Printf("Headers: %+v\n\n", header)
		}
	}()

	var transaction *chain.GetTransaction
	switch net {
	case "mainnet":
		transaction, err = btcNode.Node.BlockchainTransactionGet( /*mainnet*/
			ctx, "f53a8b83f85dd1ce2a6ef4593e67169b90aaeb402b3cf806b37afc634ef71fbc", false)
	case "testnet":
		transaction, err = btcNode.Node.BlockchainTransactionGet( /*testnet3*/
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
	// to true. Get a list of supported queries from the server itself.
	nodeSupportsAddressQueries := false
	if nodeSupportsAddressQueries {
		balance, err := btcNode.Node.BlockchainAddressGetBalance(ctx, "bc1qv5wppm0xwarzwea9xxascc5rne7c0c296h7y5p")
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
