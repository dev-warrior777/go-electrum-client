package btc

import (
	"context"
	"crypto/tls"

	"github.com/dev-warrior777/go-electrum-client/chain"
)

type BtcNode struct {
	Net  string
	Node *chain.Node
}

func NewBtcNode(net string) *BtcNode {
	n := chain.NewNode()
	bn := BtcNode{
		Net:  net,
		Node: n,
	}

	return &bn
}

// Connect connects to a single BTC Electrum server over TCP or SSL depending if
// config is empty or not.
func (bn *BtcNode) Connect(ctx context.Context, addr string, auth *tls.Config) error {
	return bn.Node.Connect(ctx, addr, auth)
}

func (bn *BtcNode) Disconnect() {
	bn.Node.Disconnect()
}
