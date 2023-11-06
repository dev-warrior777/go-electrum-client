package elxbtc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/dev-warrior777/go-electrum-client/electrumx"
)

type SingleNode struct {
	Config *electrumx.NodeConfig
	Server *electrumx.ElectrumXSvrConn
}

func NewSingleNode(cfg *electrumx.NodeConfig) *SingleNode {
	n := SingleNode{
		Config: cfg,
		Server: nil,
	}
	return &n
}

func (s *SingleNode) Start() error {
	trustedServer := s.Config.TrustedPeer
	if trustedServer == nil {
		return errors.New("SingleNode requires a trusted ElectrumX server")
	}
	netProto := trustedServer.Network()
	addr := trustedServer.String()
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		log.Fatal(err)
	}

	var tlsConfig *tls.Config = nil
	if netProto == "ssl" {
		rootCAs, _ := x509.SystemCertPool()
		tlsConfig = &tls.Config{
			InsecureSkipVerify: true,
			RootCAs:            rootCAs,
			MinVersion:         tls.VersionTLS12, // works ok
			ServerName:         host,
		}
	}

	opts := &electrumx.ConnectOpts{
		TLSConfig:   tlsConfig,
		DebugLogger: electrumx.StdoutPrinter,
	}

	network := s.Config.Params.Name
	genesis := s.Config.Params.GenesisHash.String()
	fmt.Println("starting single node on", network, "genesis", genesis)

	// Our context shared with client for cancellation
	// pro TODO:
	// ctx, cancel := context.WithCancel(context.Background())
	// dev
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)

	sc, err := electrumx.ConnectServer(ctx, addr, opts)
	if err != nil {
		cancel()
		return err
	}
	s.Server = &electrumx.ElectrumXSvrConn{
		SvrConn:   sc,
		SvrCtx:    ctx,
		SvrCancel: cancel,
		Running:   true,
	}

	fmt.Println(sc.Proto())

	fmt.Printf("\n ** Connected to %s **\n", network)

	feats, err := sc.Features(ctx)
	if err != nil {
		cancel()
		return err
	}

	if feats.Genesis != genesis {
		return errors.New("wrong genesis hash for Bitcoin")
	}
	fmt.Println("Genesis correct: ", "0x"+feats.Genesis)

	return nil
}

func (s *SingleNode) Stop() {
	fmt.Println("stopping single node...")
	if !s.Server.Running {
		fmt.Println("..not running")
		return
	}
	s.Server.Running = false
	s.Server.SvrConn.Shutdown()
	<-s.Server.SvrConn.Done()
	fmt.Println("..stopped single node")
}

func (s *SingleNode) GetServerConn() *electrumx.ElectrumXSvrConn {
	return s.Server
}

var ErrServerNotRunning error = errors.New("server not running")

func (s *SingleNode) BlockHeaders(startHeight, blockCount uint32) (*electrumx.GetBlockHeadersResult, error) {
	server := s.Server
	if !server.Running {
		return nil, ErrServerNotRunning
	}
	return server.SvrConn.BlockHeaders(server.SvrCtx, startHeight, blockCount)
}

func (s *SingleNode) SubscribeHeaders() (*electrumx.SubscribeHeadersResult, <-chan *electrumx.SubscribeHeadersResult, error) {
	server := s.Server
	if !server.Running {
		return nil, nil, ErrServerNotRunning
	}
	return server.SvrConn.SubscribeHeaders(server.SvrCtx)
}

// /////////////////////////////////////////////////////////////////////////////
// MultiNode
// //////////
type MultiNode struct {
	NodeConfig *electrumx.NodeConfig
	ServerMap  map[string]*electrumx.ElectrumXSvrConn
}

func (m *MultiNode) Start() error {
	fmt.Println("starting multi node")
	// TODO:
	return nil
}
func (m *MultiNode) Stop() {
	fmt.Println("stopping multi node")
	// TODO:
}
