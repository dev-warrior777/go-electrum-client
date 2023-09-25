package btc

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/dev-warrior777/go-electrum-client/chain"
)

type BtcNode struct {
	Net        string
	Node       *chain.Node
	pingCancel context.CancelFunc
}

func NewBtcNode(net string) *BtcNode {
	n := chain.NewNode()
	return &BtcNode{
		Net:  net,
		Node: n,
	}
}

var _ = chain.ElectrumXNode(&BtcNode{})

// Connect connects to a single BTC ElectrumX server over TCP or SSL depending if
// config is empty or not.
func (bn *BtcNode) Connect(ctx context.Context, addr string, auth *tls.Config) error {
	return bn.Node.Connect(ctx, addr, auth)
}

func (bn *BtcNode) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	bn.pingCancel = cancel
	go bn.doPing(ctx)
}

// Disconnect disconnects from the BTC ElectrumX server and does cleanup.
func (bn *BtcNode) Disconnect() {
	if bn.pingCancel != nil {
		bn.pingCancel()
	}
	bn.Node.Disconnect()
}

// doPing sends a keepalive ping to the electrumX server. Run as a goroutine
func (bn *BtcNode) doPing(ctx context.Context) {
	for {
		if err := bn.Node.Ping(ctx); err != nil {
			fmt.Println(err)
		}
		fmt.Println("---------------------------- PING -----------------------------")

		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			if chain.DebugMode {
				fmt.Println("BTC disconnect keep alive Ping")
			}
			return
		}
	}
}
