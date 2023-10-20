package client

import (
	"github.com/dev-warrior777/go-electrum-client/electrumx"
)

type Node interface {
	Start()
	Stop()
}

type SingleNode struct {
	Server electrumx.ServerConn
}

func NewSingleNode(config *Config) (*SingleNode, error) {

	return nil, nil
}

type MultiNode struct {
	ServerMap map[string]electrumx.ServerConn
}
