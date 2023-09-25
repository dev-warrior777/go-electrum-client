package chain

import (
	"context"
	"crypto/tls"
)

var DebugMode bool

type ElectrumXNode interface {
	// Connect connects to a single ElectrumX server over TCP or SSL depending if
	// config is empty or not.
	Connect(ctx context.Context, addr string, auth *tls.Config) error

	// Starts local services such as pinging ElectrumX server for keepalive.
	Start()

	// Disconnect disconnects from the ElectrumX server and does cleanup.
	Disconnect()
}
