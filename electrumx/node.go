package electrumx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
)

var ErrNotConnected = errors.New("node not connected")

type nodeState int

const (
	DISCONNECTED  nodeState = 0
	CONNECTING    nodeState = 1
	CONNECTED     nodeState = 2
	DISCONNECTING nodeState = 3
)

type Node struct {
	state               nodeState
	serverAddr          string
	connectOpts         *ConnectOpts
	server              *Server
	leader              bool
	syncingHeaders      bool
	networkHeaders      *Headers
	rcvTipChangeNotify  chan int64
	rcvScriptHashNotify chan *ScripthashStatusResult
}

func newNode(
	netAddr net.Addr,
	isLeader bool,
	networkHeaders *Headers,
	rcvTipChangeNotify chan int64,
	rcvScriptHashNotify chan *ScripthashStatusResult) (*Node, error) {

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
		server: &Server{
			conn:                 nil,
			scripthashNotifyChan: nil,
			connected:            false,
		},
		leader:              isLeader,
		syncingHeaders:      false,
		networkHeaders:      networkHeaders,
		rcvTipChangeNotify:  rcvTipChangeNotify,
		rcvScriptHashNotify: rcvScriptHashNotify,
	}
	return n, nil
}

func (n *Node) start(ctx context.Context, network, nettype, genesis string) error {
	fmt.Printf("starting a new node on %s %s - genesis: %s\n", network, nettype, genesis)
	// connect to electrumX
	n.state = CONNECTING
	sc, err := ConnectServer(ctx, n.serverAddr, n.connectOpts)
	if err != nil {
		n.state = DISCONNECTED
		return err
	}
	n.state = CONNECTED
	n.server.conn = sc
	n.server.scripthashNotifyChan = sc.GetScripthashNotify()
	n.server.connected = true
	fmt.Printf("** Connected to %s using %s **\n", nettype, sc.Proto())
	// check genesis
	feats, err := sc.Features(ctx)
	if err != nil {
		n.state = DISCONNECTED
		sc.cancel()
		return err
	}
	if feats.Genesis != genesis {
		n.state = DISCONNECTED
		sc.cancel()
		return fmt.Errorf("wrong genesis hash for %s %s", network, nettype)
	}
	// now server is connected check if we have required functions like
	// GetTransaction which is not supported on some servers.
	if !testNeededServerFns(ctx, sc, network, nettype) {
		n.state = DISCONNECTED
		sc.cancel()
		return errors.New("server does not implement needed function")
	}
	// Node is up and ready - if not leader then we exit here
	if !n.leader {
		return nil
	}
	// leader sync headers
	n.syncingHeaders = true
	err = n.syncHeaders(ctx)
	if err != nil {
		n.state = DISCONNECTED
		sc.cancel()
		return err
	}
	n.syncingHeaders = false
	n.runLeader(ctx)
	return nil
}

// promoteToLeader makes a non-leader responsible for incoming notifications
func (n *Node) promoteToLeader(ctx context.Context) error {
	n.leader = true
	n.runLeader(ctx)
	return nil
}

// run listens for incoming finding new peers
func (n *Node) runLeader(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
				// TODO: find new server peers
			}
		}
	}()
}

func testNeededServerFns(ctx context.Context, sc *ServerConn, network, nettype string) bool {
	switch network {
	case "Bitcoin":
		switch nettype {
		case "testnet", "testnet3":
			txid := "581d837b8bcca854406dc5259d1fb1e0d314fcd450fb2d4654e78c48120e0135"
			_, err := sc.GetTransaction(ctx, txid)
			if err != nil {
				return false
			}
		case "mainnet":
			txid := "f53a8b83f85dd1ce2a6ef4593e67169b90aaeb402b3cf806b37afc634ef71fbc"
			_, err := sc.GetTransaction(ctx, txid)
			if err != nil {
				return false
			}
			// ignore regtest
		}
	}
	return true
}

//-----------------------------------------------------------------------------
// Server API
//-----------------------------------------------------------------------------

func (n *Node) getHeadersNotify() <-chan *HeadersNotifyResult {
	if n.state != CONNECTED {
		return nil
	}
	return n.server.conn.GetHeadersNotify()
}

func (n *Node) subscribeHeaders(ctx context.Context) (*HeadersNotifyResult, error) {
	if n.state != CONNECTED {
		return nil, ErrNotConnected
	}
	return n.server.conn.SubscribeHeaders(ctx)
}

func (n *Node) getScripthashNotify() (<-chan *ScripthashStatusResult, error) {
	if n.state != CONNECTED {
		return nil, ErrNotConnected
	}
	return n.server.conn.GetScripthashNotify(), nil
}

func (n *Node) subscribeScripthashNotify(ctx context.Context, scripthash string) (*ScripthashStatusResult, error) {
	if n.state != CONNECTED {
		return nil, ErrNotConnected
	}
	return n.server.conn.SubscribeScripthash(ctx, scripthash)
}

func (n *Node) unsubscribeScripthashNotify(ctx context.Context, scripthash string) {
	if n.state != CONNECTED {
		return
	}
	n.server.conn.UnsubscribeScripthash(ctx, scripthash)
}

func (n *Node) blockHeader(ctx context.Context, height int64) (string, error) {
	if n.state != CONNECTED {
		return "", ErrNotConnected
	}
	return n.server.conn.BlockHeader(ctx, uint32(height))
}

func (n *Node) blockHeaders(ctx context.Context, startHeight int64, blockCount int) (*GetBlockHeadersResult, error) {
	if n.state != CONNECTED {
		return nil, ErrNotConnected
	}
	return n.server.conn.BlockHeaders(ctx, startHeight, blockCount)
}

func (n *Node) getHistory(ctx context.Context, scripthash string) (HistoryResult, error) {
	if n.state != CONNECTED {
		return nil, ErrNotConnected
	}
	return n.server.conn.GetHistory(ctx, scripthash)
}

func (n *Node) getListUnspent(ctx context.Context, scripthash string) (ListUnspentResult, error) {
	if n.state != CONNECTED {
		return nil, ErrNotConnected
	}
	return n.server.conn.GetListUnspent(ctx, scripthash)
}

func (n *Node) getTransaction(ctx context.Context, txid string) (*GetTransactionResult, error) {
	if n.state != CONNECTED {
		return nil, ErrNotConnected
	}
	return n.server.conn.GetTransaction(ctx, txid)
}

func (n *Node) getRawTransaction(ctx context.Context, txid string) (string, error) {
	if n.state != CONNECTED {
		return "", ErrNotConnected
	}
	return n.server.conn.GetRawTransaction(ctx, txid)
}

func (n *Node) broadcast(ctx context.Context, rawTx string) (string, error) {
	if n.state != CONNECTED {
		return "", ErrNotConnected
	}
	return n.server.conn.Broadcast(ctx, rawTx)
}

func (n *Node) estimateFeeRate(ctx context.Context, confTarget int64) (int64, error) {
	if n.state != CONNECTED {
		return 0, ErrNotConnected
	}
	return n.server.conn.EstimateFee(ctx, confTarget)
}
