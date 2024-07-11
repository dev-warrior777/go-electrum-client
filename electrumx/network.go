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
	"github.com/decred/dcrd/crypto/rand"
)

// network
//    |
//    leader id 0	nodeCtx0
//    |
//    peers
//      |
//       -- peer id 1	nodeCtx1
//      |
//       -- peer id 2	nodeCtx2
//      |
//       -- peer id 3	nodeCtx3
//      |
//       ...
//
// Network is the controller of all the nodes we start. Each node is started
// within it's own child context of the goele ctx. The nodeCtx of each can
// be cancelled by either the Network or the server connection on disconnect.
// Mis-behaving nodes can also cancel in rare cases such as obviously wrong
// information sent.

// Network was the cancel cause
var errNetworkCanceled = errors.New("Network Canceled")

// server-connection was the cancel cause
var errServerCanceled = errors.New("Server Canceled")

// misbehaving node server was the cancel cause
var errNodeMisbehavingCanceled = errors.New("Server Misbehaving Canceled")

var errNoNetwork = errors.New("network not started")
var errNoLeader = errors.New("no leader node is assigned - try again in 10 seconds")

var peerId uint32 // stomic

type peerNode struct {
	id         uint32
	isLeader   bool
	isTrusted  bool
	netAddr    *NodeServerAddr
	node       *Node
	nodeCtx    context.Context
	nodeCancel context.CancelCauseFunc
}

func newPeerNodeWithId(
	isLeader bool,
	isTrusted bool,
	netAddr *NodeServerAddr,
	node *Node,
	nodeCtx context.Context,
	nodeCancel context.CancelCauseFunc) *peerNode {

	newId := atomic.LoadUint32(&peerId)
	pn := &peerNode{
		id:         newId,
		isLeader:   isLeader,
		isTrusted:  isTrusted,
		netAddr:    netAddr,
		node:       node,
		nodeCtx:    nodeCtx,
		nodeCancel: nodeCancel,
	}
	atomic.AddUint32(&peerId, 1)
	return pn
}

type Network struct {
	config          *ElectrumXConfig
	started         bool
	startMtx        sync.Mutex
	leader          *peerNode
	peers           []*peerNode
	peersMtx        sync.RWMutex
	knownServers    []*serverAddr
	knownServersMtx sync.Mutex
	headers         *headers
	// static channels to client for the lifetime of the main context
	clientTipChangeNotify  chan int64
	clientScripthashNotify chan *ScripthashStatusResult
}

func NewNetwork(config *ElectrumXConfig) *Network {
	h := newHeaders(config)
	n := &Network{
		config:                 config,
		started:                false,
		leader:                 nil,
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
	// leader up and headers synced
	net.started = true
	// ask leader for it's known peers
	net.getServerPeers(ctx)
	// bootstrap peers loop with leader's connection
	go net.peersMonitor(ctx)
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
	nodeCtx, nodeCancel := context.WithCancelCause(ctx)
	err = node.start(nodeCtx, nodeCancel, network, nettype, genesis)
	if err != nil {
		nodeCancel(errNetworkCanceled)
		return err
	}
	// node is up, add to peerNodes if not leader
	peer := newPeerNodeWithId(isLeader, isTrusted, netAddr, node, nodeCtx, nodeCancel)
	if isLeader {
		net.leader = peer
	} else {
		net.addPeer(peer)
	}
	// ask our peer for it's electrumx server's Peers & load server list from file
	net.getServerPeers(ctx)
	return nil
}

// get and update a list of a known servers - not locked
func (net *Network) getServerPeers(ctx context.Context) {
	err := net.getServers(ctx)
	if err != nil {
		return
	}
}

// add a started peer to running peers - not locked
func (net *Network) addPeer(newPeer *peerNode) {
	net.peers = append(net.peers, newPeer)
}

// remove a running peer - not locked
func (net *Network) removePeer(oldPeer *peerNode) {
	nodesLen := len(net.peers)
	newPeers := make([]*peerNode, 0, nodesLen-1)
	for _, peer := range net.peers {
		if oldPeer.id == peer.id {
			fmt.Printf("removing peer %d\n", oldPeer.id)
		} else {
			newPeers = append(newPeers, peer)
		}
	}
	net.peers = newPeers
}

// getNumPeers gets the number of current peer nodes - not locked
func (net *Network) getNumPeers() int {
	return len(net.peers)
}

// getLeader gets leader node if any - not locked
func (net *Network) getLeader() *peerNode {
	return net.leader
}

// -----------------------------------------------------------------------------
// Peer nodes monitor
// -----------------------------------------------------------------------------

// bootstrap peers and monitor leader - run as goroutine
func (net *Network) peersMonitor(ctx context.Context) {
	// "The ticker will adjust the time interval or drop ticks to make up for
	// slow receivers." go doc
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			for _, peer := range net.peers {
				peer.nodeCancel(errNetworkCanceled)
			}
			return
		case <-t.C:
			net.checkLeader(ctx)
			net.reapDeadPeers()
			net.startNewPeerMaybe(ctx)
		}
	}
}

func (net *Network) checkLeader(ctx context.Context) {
	net.peersMtx.Lock()
	defer net.peersMtx.Unlock()

	leader := net.getLeader()
	if leader != nil {
		if leader.nodeCtx.Err() == nil {
			// fast path
			return
		}
		fmt.Printf("checkLeader: need a new leader for: %s nodeCtx.Err(): %v\n",
			leader.netAddr, context.Cause(leader.nodeCtx))
	} else {
		fmt.Printf("checkLeader: need a new leader - current leader is <nil>\n")
	}

	// we need a new leader

	// any running peers we can promote?
	numPeers := net.getNumPeers()
	if numPeers > 0 {
		// promote one from the list.
		for _, peer := range net.peers {
			if peer.nodeCtx.Err() != nil {
				continue
			}
			err := peer.node.promoteToLeader(peer.nodeCtx)
			if err != nil {
				fmt.Printf("checkLeader: cannot start %s %v\n", peer.netAddr, err)
				peer.nodeCancel(errNetworkCanceled)
				continue
			}
			net.leader = peer
			return
		}
	}
	// no running peers so make new leader
	net.startNewLeader(ctx)
}

func (net *Network) reapDeadPeers() {
	net.peersMtx.Lock()
	defer net.peersMtx.Unlock()

	var peersToRemove = make([]*peerNode, 0)
	for _, peer := range net.peers {
		if peer.nodeCtx.Err() != nil {
			peersToRemove = append(peersToRemove, peer)
		}
	}
	for i, peer := range peersToRemove {
		net.removePeer(peer)
		fmt.Printf("reapDeadPeers - online peers after reap (#%d) is: %d\n\n", i+1, net.getNumPeers())
	}
}

func (net *Network) startNewPeerMaybe(ctx context.Context) {
	net.peersMtx.Lock()
	defer net.peersMtx.Unlock()

	numPeers := net.getNumPeers()
	if numPeers >= net.config.MaxOnlinePeers {
		return
	}
	if len(net.knownServers) == 0 {
		fmt.Println("startNewPeerMaybe: no known servers")
		return
	}
	available := net.availableServers()
	if len(available) == 0 {
		fmt.Println("startNewPeerMaybe: no known servers that are not already connected")
		return
	}
	// start one new peer .. from the pseudo randomized list
	addr := toNetAddr(available[0])
	err := net.startNewPeer(ctx, addr, false, false) // dialerCtx time limited to 10s
	if err != nil {
		fmt.Printf(" ..cannot start %s %v\n", addr.String(), err)
		net.removeServer(available[0])
	}
	net.shufflePeers()
	fmt.Printf("startNewPeerMaybe: online peers: %d\n\n", net.getNumPeers())
}

func toNetAddr(saddr *serverAddr) *NodeServerAddr {
	return &NodeServerAddr{
		Net:  saddr.Net,
		Addr: saddr.Address,
	}
}

// build a pseudo randomized list of known servers that have not yet been started
func (net *Network) availableServers() []*serverAddr {
	var available = make([]*serverAddr, 0)
	servers := net.knownServers
	for _, server := range servers {
		if server.IsOnion {
			continue
		}
		matchedAnyNetAddr := false
		for _, peer := range net.peers {
			if peer.nodeCtx.Err() != nil {
				continue
			}
			svrNodeAddr := toNetAddr(server)
			if svrNodeAddr.IsEqual(peer.netAddr) {
				matchedAnyNetAddr = true
				break
			}
		}
		if !matchedAnyNetAddr {
			available = append(available, server)
		}
	}
	// pseudo randomize the list order
	shuffleAvailableKnownServers(available)

	return available
}

func (net *Network) startNewLeader(ctx context.Context) {
	fmt.Printf("startNewLeader\n")
	// get a free known server
	if len(net.knownServers) == 0 {
		fmt.Println("startNewLeader: no known servers")
		return
	}
	available := net.availableServers()
	if len(available) == 0 {
		fmt.Println("startNewLeader: no known servers that are not already connected")
		return
	}
	// TODO: filter servers again by reputation, capabilities and banlist

	// start one node up as new leader .. from the pseudo randomized list
	addr := toNetAddr(available[0])
	err := net.startNewPeer(ctx, addr, true, false) // dialerCtx time limited to 10s
	if err != nil {
		fmt.Printf(" ..cannot start %s %v\n", addr.String(), err)
		net.removeServer(available[0])
	}
	fmt.Printf("startNewLeader: leader (untrusted) online: %s\n\n", addr.String())
}

func (net *Network) shufflePeers() {
	numPeers := net.getNumPeers()
	fmt.Println("shufflePeers", numPeers)
	if numPeers > 1 {
		rand.ShuffleSlice(net.peers)
	}
}

func shuffleAvailableKnownServers(available []*serverAddr) {
	numServers := len(available)
	fmt.Println("shuffleAvailableKnownServers", numServers)
	if numServers > 1 {
		rand.ShuffleSlice(available)
	}
}

//-----------------------------------------------------------------------------
// API Local headers
//-----------------------------------------------------------------------------

func (net *Network) Tip() (int64, error) {
	if !net.started {
		return 0, errNoNetwork
	}
	return net.headers.getClientTip(), nil
}

func (net *Network) BlockHeader(height int64) (*wire.BlockHeader, error) {
	if !net.started {
		return nil, errNoNetwork
	}
	return net.headers.getBlockHeader(height)
}

func (net *Network) BlockHeaders(startHeight int64, blockCount int64) ([]*wire.BlockHeader, error) {
	if !net.started {
		return nil, errNoNetwork
	}
	return net.headers.getBlockHeaders(startHeight, blockCount)
}

// -----------------------------------------------------------------------------
// API Pass thru from Client
// -----------------------------------------------------------------------------

func (net *Network) SubscribeScripthashNotify(ctx context.Context, scripthash string) (*ScripthashStatusResult, error) {
	if !net.started {
		return nil, errNoNetwork
	}
	net.peersMtx.Lock()
	defer net.peersMtx.Unlock()
	leader := net.getLeader()
	if leader == nil {
		return nil, errNoLeader
	}
	return leader.node.subscribeScripthashNotify(ctx, scripthash)
}

func (net *Network) UnsubscribeScripthashNotify(ctx context.Context, scripthash string) {
	if !net.started {
		return
	}
	net.peersMtx.Lock()
	defer net.peersMtx.Unlock()
	leader := net.getLeader()
	if leader == nil {
		return
	}
	leader.node.unsubscribeScripthashNotify(ctx, scripthash)
}

func (net *Network) GetHistory(ctx context.Context, scripthash string) (HistoryResult, error) {
	if !net.started {
		return nil, errNoNetwork
	}
	net.peersMtx.Lock()
	defer net.peersMtx.Unlock()
	leader := net.getLeader()
	if leader == nil {
		return nil, errNoLeader
	}
	return leader.node.getHistory(ctx, scripthash)
}

func (net *Network) GetListUnspent(ctx context.Context, scripthash string) (ListUnspentResult, error) {
	if !net.started {
		return nil, errNoNetwork
	}
	net.peersMtx.Lock()
	defer net.peersMtx.Unlock()
	leader := net.getLeader()
	if leader == nil {
		return nil, errNoLeader
	}
	return leader.node.getListUnspent(ctx, scripthash)
}

func (net *Network) GetTransaction(ctx context.Context, txid string) (*GetTransactionResult, error) {
	if !net.started {
		return nil, errNoNetwork
	}
	net.peersMtx.Lock()
	defer net.peersMtx.Unlock()
	leader := net.getLeader()
	if leader == nil {
		return nil, errNoLeader
	}
	return leader.node.getTransaction(ctx, txid)
}

func (net *Network) GetRawTransaction(ctx context.Context, txid string) (string, error) {
	if !net.started {
		return "", errNoNetwork
	}
	net.peersMtx.Lock()
	defer net.peersMtx.Unlock()
	leader := net.getLeader()
	if leader == nil {
		return "", errNoLeader
	}
	return leader.node.getRawTransaction(ctx, txid)
}

func (net *Network) Broadcast(ctx context.Context, rawTx string) (string, error) {
	if !net.started {
		return "", errNoNetwork
	}
	net.peersMtx.Lock()
	defer net.peersMtx.Unlock()
	leader := net.getLeader()
	if leader == nil {
		return "", errNoLeader
	}
	return leader.node.broadcast(ctx, rawTx)
}

func (net *Network) EstimateFeeRate(ctx context.Context, confTarget int64) (int64, error) {
	if !net.started {
		return 0, errNoNetwork
	}
	net.peersMtx.Lock()
	defer net.peersMtx.Unlock()
	leader := net.getLeader()
	if leader == nil {
		return 0, errNoLeader
	}
	return leader.node.estimateFeeRate(ctx, confTarget)
}
