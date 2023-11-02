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
	NodeCtx       context.Context
	NodeCtxCancel context.CancelFunc
	NodeConfig    *electrumx.NodeConfig
	Server        *electrumx.ServerConn
}

func NewSingleNode(cfg *electrumx.NodeConfig) *SingleNode {
	n := SingleNode{
		NodeConfig: cfg,
		Server:     nil,
	}
	return &n
}

func (s *SingleNode) Start() error {
	network := s.NodeConfig.Params.Name
	genesis := s.NodeConfig.Params.GenesisHash.String()
	fmt.Println("starting single node on", network, "genesis", genesis)
	// dev only
	s.NodeCtx, s.NodeCtxCancel = signal.NotifyContext(context.Background(), os.Interrupt)
	// s.NodeCtx, s.NodeCtxCancel = context.WithCancel(context.Background())
	ctx := s.NodeCtx
	cancel := s.NodeCtxCancel
	trustedServer := s.NodeConfig.TrustedPeer
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

	s.Server, err = electrumx.ConnectServer(ctx, addr, opts)
	if err != nil {
		return err
	}
	sc := s.Server

	fmt.Println(sc.Proto())

	fmt.Printf("\n\n ** Connected to %s **\n\n", network)

	feats, err := sc.Features(ctx)
	if err != nil {
		cancel()
		return err
	}

	if feats.Genesis != genesis {
		cancel()
		return errors.New("wrong genesis hash for Bitcoin")
	}
	fmt.Println("Genesis correct: ", "0x"+feats.Genesis)

	return nil
}

func (s *SingleNode) Stop() {
	fmt.Println("stopping single node")
	s.NodeCtxCancel()
	fmt.Println("stopped single node")
}

type MultiNode struct {
	NodeCtx    context.Context
	NodeConfig *electrumx.NodeConfig
	ServerMap  map[string]*electrumx.ServerConn
}

func (m *MultiNode) Start() error {
	fmt.Println("starting multi node")
	// TODO:
	return nil
}

func (m *MultiNode) Stop() {
	fmt.Println("stopping single node")
	// TODO:
}
