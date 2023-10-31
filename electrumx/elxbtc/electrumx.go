package elxbtc

import (
	"fmt"

	"github.com/dev-warrior777/go-electrum-client/electrumx"
)

type SingleNode struct {
	NodeConfig *electrumx.NodeConfig
	Server     *electrumx.ServerConn
}

func NewSingleNode(cfg *electrumx.NodeConfig) *SingleNode {
	n := SingleNode{
		NodeConfig: cfg,
		Server:     nil,
	}
	return &n
}
func (s *SingleNode) Start() error {
	fmt.Println("starting single node")
	// TODO:
	return nil
}

type MultiNode struct {
	NodeConfig *electrumx.NodeConfig
	ServerMap  map[string]*electrumx.ServerConn
}

func (m *MultiNode) Start() error {
	fmt.Println("starting multi node")
	// TODO:
	return nil
}
