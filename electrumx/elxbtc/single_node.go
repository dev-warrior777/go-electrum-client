package elxbtc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/dev-warrior777/go-electrum-client/electrumx"
)

type SingleNode struct {
	started          bool
	restarting       chan *electrumx.NetworkRestart
	config           *electrumx.ElectrumXConfig
	connectOpts      *electrumx.ConnectOpts
	serverAddr       string
	scripthashNotify chan *electrumx.ScripthashStatusResult
	headersNotify    chan *electrumx.HeadersNotifyResult
	serverMtx        sync.Mutex
	server           *electrumx.Server
}

func NewSingleNode(cfg *electrumx.ElectrumXConfig) (*SingleNode, error) {
	trustedServer := cfg.TrustedPeer
	if trustedServer == nil {
		return nil, errors.New(
			"SingleNode requires a trusted ElectrumX server (in the config)")
	}
	netProto := trustedServer.Network()
	addr := trustedServer.String()
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
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
	connectOpts := &electrumx.ConnectOpts{
		TLSConfig:   tlsConfig,
		DebugLogger: electrumx.StderrPrinter,
	}

	n := SingleNode{
		started:          false,
		restarting:       make(chan *electrumx.NetworkRestart),
		config:           cfg,
		connectOpts:      connectOpts,
		serverAddr:       addr,
		scripthashNotify: make(chan *electrumx.ScripthashStatusResult, 16), // 128 bytes/slot
		headersNotify:    make(chan *electrumx.HeadersNotifyResult, 16),    // 168 bytes/slot
		server: &electrumx.Server{
			Conn:                 nil,
			ScripthashNotifyChan: nil,
			HeadersNotifyChan:    nil,
			Connected:            false,
		},
	}
	return &n, nil
}

func (s *SingleNode) Start(clientCtx context.Context) error {
	if s.started {
		return errors.New("already started")
	}
	err := s.start(clientCtx)
	if err != nil {
		return err
	}
	s.started = true
	return nil
}

func (s *SingleNode) start(clientCtx context.Context) error {
	network := s.config.Params.Name
	genesis := s.config.Params.GenesisHash.String()
	fmt.Println("starting single node on", network, "genesis", genesis)

	// connect to electrumX
	sc, err := electrumx.ConnectServer(clientCtx, s.serverAddr, s.connectOpts)
	if err != nil {
		return err
	}

	s.server.Conn = sc
	s.server.HeadersNotifyChan = sc.GetHeadersNotify()
	s.server.ScripthashNotifyChan = sc.GetScripthashNotify()
	s.server.Connected = true

	fmt.Printf("** Connected to %s using %s **\n", network, sc.Proto())

	feats, err := sc.Features(clientCtx)
	if err != nil {
		return err
	}

	if feats.Genesis != genesis {
		return errors.New("wrong genesis hash for Bitcoin")
	}

	// now server is up check if we have required functions like GetTransaction
	// which is not supported on at least one server .. maybe more.
	switch network {
	case "testnet", "testnet3":
		txid := "581d837b8bcca854406dc5259d1fb1e0d314fcd450fb2d4654e78c48120e0135"
		_, err := sc.GetTransaction(clientCtx, txid)
		if err != nil {
			return err
		}
	case "mainnet":
		txid := "f53a8b83f85dd1ce2a6ef4593e67169b90aaeb402b3cf806b37afc634ef71fbc"
		_, err := sc.GetTransaction(clientCtx, txid)
		if err != nil {
			return err
		}
		// ignore regtest
	}

	go s.run(clientCtx)

	return nil
}

func (s *SingleNode) run(clientCtx context.Context) {

	// Monitor connection loop

	for {
	newServer:
		for {
			select {
			case <-clientCtx.Done():
				return
			case <-s.server.Conn.Done():
				s.serverMtx.Lock()
				s.server.Connected = false
				s.serverMtx.Unlock()
				break newServer
			case hdrs := <-s.server.HeadersNotifyChan:
				if hdrs != nil && s.serverRunning() {
					s.headersNotify <- hdrs
				}
			case status := <-s.server.ScripthashNotifyChan:
				if status != nil && s.serverRunning() {
					s.scripthashNotify <- status
				}
			}
		}

		fmt.Println("disconnected: will try a new connection in 5 sec")

		for {
			time.Sleep(5 * time.Second)
			fmt.Println("trying to make a new connection")

			// connect to electrumX
			sc, err := electrumx.ConnectServer(clientCtx, s.serverAddr, s.connectOpts)
			if err == nil {
				s.serverMtx.Lock()
				s.server.Conn = sc
				s.server.HeadersNotifyChan = sc.GetHeadersNotify()
				s.server.ScripthashNotifyChan = sc.GetScripthashNotify()
				s.server.Connected = true
				// notify client to resubscribe to headers and scripthashes
				s.restarting <- &electrumx.NetworkRestart{
					Time: time.Now(),
				}
				s.serverMtx.Unlock()
				break
			}
		}
	}
}

func (s *SingleNode) RegisterNetworkRestart() <-chan *electrumx.NetworkRestart {
	return s.restarting
}

func (s *SingleNode) Stop() {
	fmt.Println("stopping single node...")
	close(s.restarting)
	close(s.headersNotify)
	close(s.scripthashNotify)
	if !s.serverRunning() {
		fmt.Println("..server not running")
		return
	}
	s.server.Conn.Shutdown()
	<-s.server.Conn.Done()
	fmt.Println("..stopped server")
}

// var ErrServerNotRunning error = errors.New("server not running")

func (s *SingleNode) serverRunning() bool {
	s.serverMtx.Lock()
	defer s.serverMtx.Unlock()
	return s.server.Connected
}

func (s *SingleNode) GetHeadersNotify() (<-chan *electrumx.HeadersNotifyResult, error) {
	s.serverMtx.Lock()
	defer s.serverMtx.Unlock()
	if !s.server.Connected {
		return nil, ErrServerNotRunning
	}
	return s.headersNotify, nil
}

func (s *SingleNode) SubscribeHeaders(ctx context.Context) (*electrumx.HeadersNotifyResult, error) {
	if !s.serverRunning() {
		return nil, ErrServerNotRunning
	}
	return s.server.Conn.SubscribeHeaders(ctx)
}

func (s *SingleNode) GetScripthashNotify() (<-chan *electrumx.ScripthashStatusResult, error) {
	s.serverMtx.Lock()
	defer s.serverMtx.Unlock()
	if !s.server.Connected {
		return nil, ErrServerNotRunning
	}
	return s.scripthashNotify, nil
}

func (s *SingleNode) SubscribeScripthashNotify(ctx context.Context, scripthash string) (*electrumx.ScripthashStatusResult, error) {
	if !s.serverRunning() {
		return nil, ErrServerNotRunning
	}
	return s.server.Conn.SubscribeScripthash(ctx, scripthash)
}

func (s *SingleNode) UnsubscribeScripthashNotify(ctx context.Context, scripthash string) {
	if !s.serverRunning() {
		return
	}
	s.server.Conn.UnsubscribeScripthash(ctx, scripthash)
}

func (s *SingleNode) BlockHeader(ctx context.Context, height int64) (string, error) {
	if !s.serverRunning() {
		return "", ErrServerNotRunning
	}
	return s.server.Conn.BlockHeader(ctx, uint32(height))
}

func (s *SingleNode) BlockHeaders(ctx context.Context, startHeight int64, blockCount int) (*electrumx.GetBlockHeadersResult, error) {
	if !s.serverRunning() {
		return nil, ErrServerNotRunning
	}
	return s.server.Conn.BlockHeaders(ctx, startHeight, blockCount)
}

func (s *SingleNode) GetHistory(ctx context.Context, scripthash string) (electrumx.HistoryResult, error) {
	if !s.serverRunning() {
		return nil, ErrServerNotRunning
	}
	return s.server.Conn.GetHistory(ctx, scripthash)
}

func (s *SingleNode) GetListUnspent(ctx context.Context, scripthash string) (electrumx.ListUnspentResult, error) {
	if !s.serverRunning() {
		return nil, ErrServerNotRunning
	}
	return s.server.Conn.GetListUnspent(ctx, scripthash)
}

func (s *SingleNode) GetTransaction(ctx context.Context, txid string) (*electrumx.GetTransactionResult, error) {
	if !s.serverRunning() {
		return nil, ErrServerNotRunning
	}
	return s.server.Conn.GetTransaction(ctx, txid)
}

func (s *SingleNode) GetRawTransaction(ctx context.Context, txid string) (string, error) {
	if !s.serverRunning() {
		return "", ErrServerNotRunning
	}
	return s.server.Conn.GetRawTransaction(ctx, txid)
}

func (s *SingleNode) Broadcast(ctx context.Context, rawTx string) (string, error) {
	if !s.serverRunning() {
		return "", ErrServerNotRunning
	}
	return s.server.Conn.Broadcast(ctx, rawTx)
}

func (s *SingleNode) EstimateFeeRate(ctx context.Context, confTarget int64) (int64, error) {
	if !s.serverRunning() {
		return 0, ErrServerNotRunning
	}
	return s.server.Conn.EstimateFee(ctx, confTarget)
}

// /////////////////////////////////////////////////////////////////////////////
// MultiNode
// //////////
type MultiNode struct {
	// nodeConfig *electrumx.NodeConfig
	// serverMap  map[string]*electrumx.ServerConn
}

func (m *MultiNode) Start(ctx context.Context) error {
	// TODO:
	return nil
}
func (m *MultiNode) Stop() {
	// TODO:
}
