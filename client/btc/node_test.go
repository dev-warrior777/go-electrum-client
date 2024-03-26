package btc

import (
	"context"
	"fmt"
	"path"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/dev-warrior777/go-electrum-client/client"
	"github.com/dev-warrior777/go-electrum-client/electrumx"
)

//maybe make livetest

// TestNodeCreate makes a single-node that connects to a testnet server
func TestNodeCreate(t *testing.T) {
	cfg := client.NewDefaultConfig()
	cfg.Testing = true
	cfg.Params = &chaincfg.TestNet3Params
	cfg.DataDir = path.Join(cfg.DataDir, "btc/testnet")
	cfg.TrustedPeer = electrumx.ServerAddr{
		Net: "ssl", Addr: "testnet.aranguren.org:51002",
		// Net: "tcp", Addr: "testnet.aranguren.org:51001",
		// Net: "ssl", Addr: "blockstream.info:993",
		// Net: "tcp", Addr: "blockstream.info:143",
	}
	c := NewBtcElectrumClient(cfg)
	ec, ok := c.(*BtcElectrumClient)
	if !ok {
		t.Fatal("client is not a *BtcElectrumClient")
	}
	ec.createNode(client.SingleNode)
	err := ec.Node.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	n := c.GetNode()
	if n == nil {
		t.Fatal("node is nil")
	}
	n.Stop()
}

func TestMultiNodeCreate(t *testing.T) {
	fmt.Println("TBD:")
}
