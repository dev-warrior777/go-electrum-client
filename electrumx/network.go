package electrumx

// Implements failover for ElectrumX servers

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/wire"
)

const (
	MAX_ONLINE_PEERS = 3 // TODO: this should be based on coin/net
	MIN_ONLINE_PEERS = 2
)

var ErrNoNetwork = errors.New("network not started")
var ErrNoLeader = errors.New("no leader node is assigned")

var peerId uint32 // stomic

type peerNode struct {
	id         uint32
	isLeader   bool
	isTrusted  bool
	netAddr    *NodeServerAddr
	node       *Node
	nodeCancel context.CancelFunc
}

func newPeerNodeWithId(isLeader, isTrusted bool, netAddr *NodeServerAddr, node *Node, nodeCancel context.CancelFunc) *peerNode {
	newId := atomic.LoadUint32(&peerId)
	pn := &peerNode{
		id:         newId,
		isLeader:   isLeader,
		isTrusted:  isTrusted,
		netAddr:    netAddr,
		node:       node,
		nodeCancel: nodeCancel,
	}
	atomic.AddUint32(&peerId, 1)
	return pn
}

type Network struct {
	config          *ElectrumXConfig
	started         bool
	startMtx        sync.Mutex
	peers           []*peerNode
	peersMtx        sync.RWMutex
	knownServers    []*serverAddr
	knownServersMtx sync.Mutex
	headers         *Headers
	// static channels to client for the lifetime of the main context
	clientTipChangeNotify  chan int64
	clientScripthashNotify chan *ScripthashStatusResult
}

func NewNetwork(config *ElectrumXConfig) *Network {
	h := NewHeaders(config)
	n := &Network{
		config:                 config,
		started:                false,
		peers:                  make([]*peerNode, 0, 10),
		knownServers:           make([]*serverAddr, 0, 30),
		headers:                h,
		clientTipChangeNotify:  make(chan int64), // unbuffered
		clientScripthashNotify: make(chan *ScripthashStatusResult, 16),
	}
	return n
}

// GetTipChangeNotify returns a channel to client to receive tip change
// notifications from the current leader node
func (net *Network) GetTipChangeNotify() <-chan int64 {
	return net.clientTipChangeNotify
}

// GetScripthashNotify returns a channel to client to receive scripthash
// notifications from the current leader node
func (net *Network) GetScripthashNotify() <-chan *ScripthashStatusResult {
	return net.clientScripthashNotify
}

func (net *Network) Start(ctx context.Context) error {
	numLoaded, err := net.loadKnownServers()
	if err != nil {
		return err
	}
	fmt.Printf("loaded %d known servers from file\n", numLoaded)
	if net.config.TrustedPeer == nil {
		// TODO: start a stored server if no trusted peer
		return errors.New("a trusted peer is required in config")
	}
	serverAddress := net.config.TrustedPeer
	net.startMtx.Lock()
	defer net.startMtx.Unlock()
	if net.started {
		return errors.New("network already started")
	}
	return net.start(ctx, serverAddress)
}

// start starts the network with one leader peer - locked under startMtx
func (net *Network) start(ctx context.Context, startServer *NodeServerAddr) error {
	// start from our trusted node as leader
	err := net.startNewPeer(ctx, startServer, true, true)
	if err != nil {
		return err
	}
	// bootstrap peers loop with leader's connection
	go net.peersMonitor(ctx)
	// leader up and headers synced
	net.started = true
	return nil
}

// startNewPeer starts up a new peer and adds to peersList - not locked
func (net *Network) startNewPeer(ctx context.Context, netAddr *NodeServerAddr, isLeader, isTrusted bool) error {
	node, err := newNode(
		netAddr,
		isLeader,
		net.headers,
		net.clientTipChangeNotify,
		net.clientScripthashNotify)
	if err != nil {
		return err
	}
	network := net.config.Chain.String()
	nettype := net.config.Params.Name
	genesis := net.config.Params.GenesisHash.String()
	// node runs in a new child context
	nodeCtx, nodeCancel := context.WithCancel(ctx)
	err = node.start(nodeCtx, network, nettype, genesis)
	if err != nil {
		nodeCancel()
		return err
	}
	// node is up, add to peerNodes
	peer := newPeerNodeWithId(isLeader, isTrusted, netAddr, node, nodeCancel)
	net.addPeer(peer)
	// ask peer for it's electrumx server's Peers & load server list from file
	net.getPeerServers(ctx)
	return nil
}

// get and update a list of a known servers - not locked
func (net *Network) getPeerServers(ctx context.Context) {
	err := net.getServers(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}
}

// add a started peer to running peers - not locked
func (net *Network) addPeer(newPeer *peerNode) {
	net.peers = append(net.peers, newPeer)
}

// remove and cancel a running peer - not locked
func (net *Network) removePeer(oldPeer *peerNode) error {
	nodesLen := len(net.peers)
	currPeers := net.peers
	newPeers := make([]*peerNode, 0, nodesLen-1)
	for _, peer := range currPeers {
		if oldPeer.id == peer.id {
			oldPeer.nodeCancel()
		} else {
			newPeers = append(newPeers, peer)
		}
	}
	net.peers = newPeers
	return nil
}

// getNumPeers gets the number of current peer nodes - not locked
func (net *Network) getNumPeers() int {
	return len(net.peers)
}

// getLeader gets leader node if any - not locked
func (net *Network) getLeader() *peerNode {
	for _, peer := range net.peers {
		if peer.isLeader {
			return peer
		}
	}
	return nil
}

// -----------------------------------------------------------------------------
// Peer nodes monitor
// -----------------------------------------------------------------------------

// bootstrap peers and monitor leader - run as goroutine
func (net *Network) peersMonitor(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			fmt.Println("ctx.Done in nodesMonitor - exiting thread")
			for _, peer := range net.peers {
				peer.nodeCancel()
			}
			return
		case <-t.C:
			net.checkLeader(ctx)
			net.startNewPeersIfRequired(ctx)
		}
	}
}

func (net *Network) checkLeader(ctx context.Context) {
	net.peersMtx.Lock()
	defer net.peersMtx.Unlock()
	leader := net.getLeader()
	if leader != nil {
		state := leader.node.getState()
		if state == CONNECTED || state == CONNECTING {
			return
		}
	}

	// we need a new leader

	// if leader still exists remove from peers list
	if leader != nil {
		net.removePeer(leader)
	}

	// any running nodes we can promote?
	numPeers := net.getNumPeers()
	if numPeers >= 1 {
		// promote first in the list TODO: reputation score in network_servers
		newLleader := net.peers[0]
		newLleader.node.promoteToLeader()
		return
	}

	// no running nodes so make new leader
	net.startNewLeader(ctx)
}

func (net *Network) startNewLeader(ctx context.Context) {
	// TODO: get a free known server and start it up as new leader
}

func (net *Network) promoteOnePeerAsNewLeader(ctx context.Context) {
	// TODO: promote one currently running peer as new leader
}

func (net *Network) startNewPeersIfRequired(ctx context.Context) {
	net.peersMtx.Lock()
	defer net.peersMtx.Unlock()
	numPeers := net.getNumPeers()
	if numPeers >= MAX_ONLINE_PEERS {
		return
	}
	net.startNewPeers(ctx)
}

func (net *Network) startNewPeers(ctx context.Context) {
	toNetAddr := func(sa *serverAddr) *NodeServerAddr {
		return &NodeServerAddr{
			Net:  sa.Net,
			Addr: sa.Address,
		}
	}

	servers := net.knownServers
	if len(servers) == 0 {
		fmt.Println("startNewPeers: no known servers")
		return
	}
	peers := net.peers
	numPeers := len(peers)
	// build a map of known servers that can be started
	var possible = make(map[*serverAddr]bool) // [server]started
	for _, server := range servers {
		if server.IsOnion {
			continue
		}
		matchedAnyNetAddr := false
		for _, peer := range peers {
			svrNodeAddr := toNetAddr(server)
			if svrNodeAddr.IsEqual(peer.netAddr) {
				matchedAnyNetAddr = true
				break
			}
		}
		if !matchedAnyNetAddr {
			possible[server] = false
		}
	}
	if len(possible) == 0 {
		fmt.Println("startNewPeers: no known servers that are not already connected")
		return
	}

	numTostart := MAX_ONLINE_PEERS - numPeers

	var peersStarted int = 0
	for i := 0; i < numTostart; i++ {
		for server, hasStarted := range possible {
			if hasStarted {
				continue
			}
			possible[server] = true
			addr := toNetAddr(server)
			err := net.startNewPeer(ctx, addr, false, false)
			if err != nil {
				fmt.Printf(" ..cannot start %s %v\n", addr.String(), err)
				continue
			}
			peersStarted++
		}
	}
	fmt.Printf("started %d new peers\n", peersStarted)
}

// -----------------------------------------------------------------------------
// Local peers
// -----------------------------------------------------------------------------
//-----------------------------------------------------------------------------
// API Local headers
//-----------------------------------------------------------------------------

func (net *Network) Tip() (int64, error) {
	if !net.started {
		return 0, ErrNoNetwork
	}
	return net.headers.getTip(), nil
}

func (net *Network) BlockHeader(height int64) (*wire.BlockHeader, error) {
	if !net.started {
		return nil, ErrNoNetwork
	}
	return net.headers.getBlockHeader(height)
}

func (net *Network) BlockHeaders(startHeight int64, blockCount int64) ([]*wire.BlockHeader, error) {
	if !net.started {
		return nil, ErrNoNetwork
	}
	return net.headers.getBlockHeaders(startHeight, blockCount)
}

// -----------------------------------------------------------------------------
// API Pass thru
// -----------------------------------------------------------------------------

func (net *Network) SubscribeScripthashNotify(ctx context.Context, scripthash string) (*ScripthashStatusResult, error) {
	if !net.started {
		return nil, ErrNoNetwork
	}
	net.peersMtx.Lock()
	defer net.peersMtx.Unlock()
	if net.getLeader() == nil {
		return nil, ErrNoLeader
	}
	return net.getLeader().node.subscribeScripthashNotify(ctx, scripthash)
}

func (net *Network) UnsubscribeScripthashNotify(ctx context.Context, scripthash string) {
	if !net.started {
		return
	}
	net.peersMtx.Lock()
	defer net.peersMtx.Unlock()
	if net.getLeader() == nil {
		return
	}
	net.getLeader().node.unsubscribeScripthashNotify(ctx, scripthash)
}

func (net *Network) GetHistory(ctx context.Context, scripthash string) (HistoryResult, error) {
	if !net.started {
		return nil, ErrNoNetwork
	}
	net.peersMtx.Lock()
	defer net.peersMtx.Unlock()
	if net.getLeader() == nil {
		return nil, ErrNoLeader
	}
	return net.getLeader().node.getHistory(ctx, scripthash)
}

func (net *Network) GetListUnspent(ctx context.Context, scripthash string) (ListUnspentResult, error) {
	if !net.started {
		return nil, ErrNoNetwork
	}
	net.peersMtx.Lock()
	defer net.peersMtx.Unlock()
	if net.getLeader() == nil {
		return nil, ErrNoLeader
	}
	return net.getLeader().node.getListUnspent(ctx, scripthash)
}

func (net *Network) GetTransaction(ctx context.Context, txid string) (*GetTransactionResult, error) {
	if !net.started {
		return nil, ErrNoNetwork
	}
	net.peersMtx.Lock()
	defer net.peersMtx.Unlock()
	if net.getLeader() == nil {
		return nil, ErrNoLeader
	}
	return net.getLeader().node.getTransaction(ctx, txid)
}

func (net *Network) GetRawTransaction(ctx context.Context, txid string) (string, error) {
	if !net.started {
		return "", ErrNoNetwork
	}
	net.peersMtx.Lock()
	defer net.peersMtx.Unlock()
	if net.getLeader() == nil {
		return "", ErrNoLeader
	}
	return net.getLeader().node.getRawTransaction(ctx, txid)
}

func (net *Network) Broadcast(ctx context.Context, rawTx string) (string, error) {
	if !net.started {
		return "", ErrNoNetwork
	}
	net.peersMtx.Lock()
	defer net.peersMtx.Unlock()
	if net.getLeader() == nil {
		return "", ErrNoLeader
	}
	return net.getLeader().node.broadcast(ctx, rawTx)
}

func (net *Network) EstimateFeeRate(ctx context.Context, confTarget int64) (int64, error) {
	if !net.started {
		return 0, ErrNoNetwork
	}
	net.peersMtx.Lock()
	defer net.peersMtx.Unlock()
	if net.getLeader() == nil {
		return 0, ErrNoLeader
	}
	return net.getLeader().node.estimateFeeRate(ctx, confTarget)
}
