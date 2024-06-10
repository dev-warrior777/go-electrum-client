package electrumx

import (
	"context"
)

type Node struct {
	// started          bool
	// stopping         bool
	// config           *ElectrumXConfig
	// connectOpts      *ConnectOpts
	// serverAddr       string
	// serverMtx        sync.Mutex
	// server           *Server
}

func (m *Node) Start(ctx context.Context) error {
	// TODO:
	return nil
}

func (m *Node) Stop() {
	// TODO:
}
