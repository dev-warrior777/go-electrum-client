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
	session             *session
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
			conn:      nil,
			connected: false,
		},
		leader:              isLeader,
		syncingHeaders:      false,
		networkHeaders:      networkHeaders,
		rcvTipChangeNotify:  rcvTipChangeNotify,
		rcvScriptHashNotify: rcvScriptHashNotify,
		session:             nil,
	}
	return n, nil
}

func (n *Node) start(nodeCtx context.Context, network, nettype, genesis string) error {
	fmt.Printf("starting a new node on %s %s - genesis: %s\n", network, nettype, genesis)
	// connect to electrumX
	n.state = CONNECTING
	sc, err := ConnectServer(nodeCtx, n.serverAddr, n.connectOpts)
	if err != nil {
		n.state = DISCONNECTED
		return err
	}
	n.state = CONNECTED
	n.server.conn = sc
	n.server.connected = true
	fmt.Printf("** Connected to %s using %s **\n", nettype, sc.Proto())
	// check genesis
	feats, err := sc.Features(nodeCtx)
	if err != nil {
		n.state = DISCONNECTED
		sc.cancel()
		<-sc.Done()
		return err
	}
	if feats.Genesis != genesis {
		n.state = DISCONNECTED
		sc.cancel()
		<-sc.Done()
		return fmt.Errorf("wrong genesis hash for %s %s", network, nettype)
	}
	// now server is connected check if we have required functions like
	// GetTransaction which is not supported on some servers.
	if !testNeededServerFns(nodeCtx, sc, network, nettype) {
		n.state = DISCONNECTED
		sc.cancel()
		<-sc.Done()
		return errors.New("server does not implement needed function")
	}
	// start a new session for this node to monitor resource use
	n.session = newSession()
	n.session.start(nodeCtx)
	// Node is up and ready - if not leader then we exit here with no session started
	if !n.leader {
		return nil
	}

	// leader sync headers
	n.syncingHeaders = true
	err = n.syncHeaders(nodeCtx)
	if err != nil {
		n.state = DISCONNECTED
		sc.cancel()
		<-sc.Done()
		return err
	}
	// sync headers
	n.syncingHeaders = false
	// start header notifications
	err = n.headersNotify(nodeCtx)
	if err != nil {
		n.state = DISCONNECTED
		sc.cancel()
		<-sc.Done()
		return err
	}

	err = n.scriptHashNotify(nodeCtx)
	if err != nil {
		n.state = DISCONNECTED
		sc.cancel()
		<-sc.Done()
		return err
	}
	return nil
}

// // promoteToLeader makes a non-leader responsible for incoming notifications
// func (n *Node) promoteToLeader(nodeCtx context.Context) error {
// 	n.leader = true
// 	return nil
// }

//-----------------------------------------------------------------------------
// Server API
//-----------------------------------------------------------------------------

func (n *Node) getHeadersNotify() chan *HeadersNotifyResult {
	if n.state != CONNECTED {
		return nil
	}
	return n.server.conn.GetHeadersNotify()
}

func (n *Node) subscribeHeaders(nodeCtx context.Context) (*HeadersNotifyResult, error) {
	if n.state != CONNECTED {
		return nil, ErrNotConnected
	}
	return n.server.conn.SubscribeHeaders(nodeCtx)
}

func (n *Node) getScripthashNotify() chan *ScripthashStatusResult {
	if n.state != CONNECTED {
		return nil
	}
	return n.server.conn.GetScripthashNotify()
}

func (n *Node) subscribeScripthashNotify(nodeCtx context.Context, scripthash string) (*ScripthashStatusResult, error) {
	if n.state != CONNECTED {
		return nil, ErrNotConnected
	}
	return n.server.conn.SubscribeScripthash(nodeCtx, scripthash)
}

func (n *Node) unsubscribeScripthashNotify(nodeCtx context.Context, scripthash string) {
	if n.state != CONNECTED {
		return
	}
	n.server.conn.UnsubscribeScripthash(nodeCtx, scripthash)
}

func (n *Node) blockHeader(nodeCtx context.Context, height int64) (string, error) {
	if n.state != CONNECTED {
		return "", ErrNotConnected
	}
	blkHdr, err := n.server.conn.BlockHeader(nodeCtx, uint32(height))
	if err == nil {
		n.session.bumpCostString(blkHdr)
	} else {
		n.session.bumpCostError()
	}
	return blkHdr, err
}

func (n *Node) blockHeaders(nodeCtx context.Context, startHeight int64, blockCount int) (*GetBlockHeadersResult, error) {
	if n.state != CONNECTED {
		return nil, ErrNotConnected
	}
	gbh_res, err := n.server.conn.BlockHeaders(nodeCtx, startHeight, blockCount)
	if err == nil {
		n.session.bumpCostString(gbh_res.HexConcat)
	} else {
		n.session.bumpCostError()
	}
	return gbh_res, err
}

func (n *Node) getHistory(nodeCtx context.Context, scripthash string) (HistoryResult, error) {
	if n.state != CONNECTED {
		return nil, ErrNotConnected
	}
	gh_res, err := n.server.conn.GetHistory(nodeCtx, scripthash)
	if err == nil {
		n.session.bumpCostStruct(gh_res)
	} else {
		n.session.bumpCostError()
	}
	return gh_res, err
}

func (n *Node) getListUnspent(nodeCtx context.Context, scripthash string) (ListUnspentResult, error) {
	if n.state != CONNECTED {
		return nil, ErrNotConnected
	}
	lu_res, err := n.server.conn.GetListUnspent(nodeCtx, scripthash)
	if err == nil {
		n.session.bumpCostStruct(lu_res)

	} else {
		n.session.bumpCostError()
	}
	return lu_res, err
}

func (n *Node) getTransaction(nodeCtx context.Context, txid string) (*GetTransactionResult, error) {
	if n.state != CONNECTED {
		return nil, ErrNotConnected
	}
	gtx_res, err := n.server.conn.GetTransaction(nodeCtx, txid)
	if err == nil {
		n.session.bumpCostString(txid)
		n.session.bumpCostStruct(gtx_res)
	} else {
		n.session.bumpCostError()
	}
	return gtx_res, err
}

func (n *Node) getRawTransaction(nodeCtx context.Context, txid string) (string, error) {
	if n.state != CONNECTED {
		return "", ErrNotConnected
	}
	grt_res, err := n.server.conn.GetRawTransaction(nodeCtx, txid)
	if err == nil {
		n.session.bumpCostString(txid)
		n.session.bumpCostString(grt_res)
	} else {
		n.session.bumpCostError()
	}
	return grt_res, err
}

func (n *Node) broadcast(nodeCtx context.Context, rawTx string) (string, error) {
	if n.state != CONNECTED {
		return "", ErrNotConnected
	}
	txid, err := n.server.conn.Broadcast(nodeCtx, rawTx)
	if err == nil {
		n.session.bumpCostString(rawTx)
		n.session.bumpCostString(txid)
	} else {
		n.session.bumpCostError()
	}
	return txid, err
}

func (n *Node) estimateFeeRate(nodeCtx context.Context, confTarget int64) (int64, error) {
	if n.state != CONNECTED {
		return 0, ErrNotConnected
	}
	ef_res, err := n.server.conn.EstimateFee(nodeCtx, confTarget)
	if err == nil {
		n.session.bumpCostBytes(8) // 2* int64
	} else {
		n.session.bumpCostError()
	}
	return ef_res, err
}
