package electrumx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
)

type nodeState int

const (
	DISCONNECTED  nodeState = 0
	CONNECTING    nodeState = 1
	CONNECTED     nodeState = 2
	DISCONNECTING nodeState = 3
)

type Node struct {
	state       nodeState
	serverAddr  string
	connectOpts *ConnectOpts
	server      *Server
}

func newNode(netAddr net.Addr) (*Node, error) {
	netProto := netAddr.Network()
	addr := netAddr.String()
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	var tlsConfig *tls.Config
	switch netProto {
	case "ssl":
		rootCAs, _ := x509.SystemCertPool()
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
			RootCAs:            rootCAs,
			MinVersion:         tls.VersionTLS12, // works ok
			ServerName:         host,
		}
	case "tcp":
		tlsConfig = nil
	default:
		return nil, fmt.Errorf("unknown protocol: %s", netProto)
	}
	connectOpts := &ConnectOpts{
		TLSConfig:   tlsConfig,
		DebugLogger: StderrPrinter,
	}

	n := &Node{
		state:       DISCONNECTED,
		connectOpts: connectOpts,
		serverAddr:  addr,
		// scripthashNotify: make(chan *electrumx.ScripthashStatusResult, 16), // 128 bytes/slot
		// headersNotify:    make(chan *electrumx.HeadersNotifyResult, 16),    // 168 bytes/slot
		server: &Server{
			Conn:                 nil,
			ScripthashNotifyChan: nil,
			HeadersNotifyChan:    nil,
			Connected:            false,
		},
	}
	return n, nil
}

func (n *Node) start(ctx context.Context, network, nettype, genesis string) error {
	fmt.Printf("starting a new node on %s %s - genesis: %s\n", network, nettype, genesis)
	// connect to electrumX
	n.state = CONNECTING
	sc, err := ConnectServer(ctx, n.serverAddr, n.connectOpts)
	if err != nil {
		return err
	}
	n.state = CONNECTED
	n.server.Conn = sc
	n.server.HeadersNotifyChan = sc.GetHeadersNotify()
	n.server.ScripthashNotifyChan = sc.GetScripthashNotify()
	n.server.Connected = true
	fmt.Printf("** Connected to %s using %s **\n", nettype, sc.Proto())
	// check genesis
	feats, err := sc.Features(ctx)
	if err != nil {
		sc.cancel()
		n.state = DISCONNECTED
		return err
	}
	if feats.Genesis != genesis {
		sc.cancel()
		n.state = DISCONNECTED
		return fmt.Errorf("wrong genesis hash for %s %s", network, nettype)
	}
	// now server is connected check if we have required functions like
	// GetTransaction which is not supported on some servers.
	switch network {
	case "Bitcoin":
		switch nettype {
		case "testnet", "testnet3":
			txid := "581d837b8bcca854406dc5259d1fb1e0d314fcd450fb2d4654e78c48120e0135"
			_, err := sc.GetTransaction(ctx, txid)
			if err != nil {
				return err
			}
		case "mainnet":
			txid := "f53a8b83f85dd1ce2a6ef4593e67169b90aaeb402b3cf806b37afc634ef71fbc"
			_, err := sc.GetTransaction(ctx, txid)
			if err != nil {
				return err
			}
			// ignore regtest
		}
	}

	return nil
}

func (n *Node) run(ctx context.Context) error {

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			}
			// TODO:
		}
	}()

	return nil
}
