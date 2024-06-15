package electrumx

// Implements failover for ElectrumX servers

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/btcsuite/btcd/wire"
)

var ErrNoNetwork = errors.New("network not started")
var ErrNoLeader = errors.New("no leader node is assigned")

var nodeId uint32

type MultiNode struct {
	id      uint32
	trusted bool
	node    *Node
}

func newMultiNodeWithId(trusted bool, node *Node) *MultiNode {
	newId := atomic.LoadUint32(&nodeId)
	m := &MultiNode{
		id:      newId,
		trusted: trusted,
		node:    node,
	}
	atomic.AddUint32(&nodeId, 1)
	return m
}

type Network struct {
	config              *ElectrumXConfig
	started             bool
	startMtx            sync.Mutex
	nodes               []*MultiNode
	leader              *MultiNode
	nodesMtx            sync.Mutex
	headers             *Headers
	rcvTipChangeNotify  chan int64
	rcvScripthashNotify chan *ScripthashStatusResult
}

func NewNetwork(config *ElectrumXConfig) *Network {
	h := NewHeaders(config)
	n := &Network{
		config:              config,
		started:             false,
		nodes:               make([]*MultiNode, 0),
		leader:              nil,
		headers:             h,
		rcvTipChangeNotify:  make(chan int64, 1),
		rcvScripthashNotify: make(chan *ScripthashStatusResult, 16),
	}
	return n
}

func (net *Network) GetTipChangeNotify() <-chan int64 {
	return net.rcvTipChangeNotify
}

func (net *Network) GetScripthashNotify() <-chan *ScripthashStatusResult {
	return net.rcvScripthashNotify
}

func (net *Network) Start(ctx context.Context) error {
	if net.config.TrustedPeer == nil {
		return errors.New("a trusted peer is required in config")
	}
	net.startMtx.Lock()
	defer net.startMtx.Unlock()
	if net.started {
		return errors.New("network already started")
	}
	return net.start(ctx)
}

func (net *Network) start(ctx context.Context) error {
	// start from our trusted node as leader
	net.nodesMtx.Lock()
	defer net.nodesMtx.Unlock()
	node, err := newNode(
		net.config.TrustedPeer, true, net.headers, net.rcvTipChangeNotify, net.rcvScripthashNotify)
	if err != nil {
		return err
	}
	network := net.config.Chain.String()
	nettype := net.config.Params.Name
	genesis := net.config.Params.GenesisHash.String()
	err = node.start(ctx, network, nettype, genesis)
	if err != nil {
		return err
	}
	m := newMultiNodeWithId(true, node)
	net.addPeer(m)
	net.leader = m

	// TODO: ...bootstrap peers loop with leader

	net.started = true
	return nil
}

func (net *Network) addPeer(m *MultiNode) {
	net.nodes = append(net.nodes, m)
}

func (net *Network) removePeer(m *MultiNode) error {
	nodesLen := len(net.nodes)
	if nodesLen < 2 {
		return fmt.Errorf("cannot remove peer node - ony have %d node(s) in peer list", nodesLen)
	}
	currNodes := net.nodes
	newNodes := make([]*MultiNode, 0, nodesLen-1)
	for _, multi := range currNodes {
		if m.id != multi.id {
			newNodes = append(newNodes, multi)
		}
	}
	net.nodes = newNodes
	if net.leader.id == m.id {
		// make a new random leader - there are 1 or more nodes in the slice
		net.leader = newNodes[0]
	}
	return nil
}

// -----------------------------------------------------------------------------
// Nodes monitor
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
	net.nodesMtx.Lock()
	defer net.nodesMtx.Unlock()
	if net.leader == nil {
		return nil, ErrNoLeader
	}
	return net.leader.node.subscribeScripthashNotify(ctx, scripthash)
}

func (net *Network) UnsubscribeScripthashNotify(ctx context.Context, scripthash string) {
	if !net.started {
		return
	}
	net.nodesMtx.Lock()
	defer net.nodesMtx.Unlock()
	net.leader.node.unsubscribeScripthashNotify(ctx, scripthash)
}

func (net *Network) GetHistory(ctx context.Context, scripthash string) (HistoryResult, error) {
	if !net.started {
		return nil, ErrNoNetwork
	}
	net.nodesMtx.Lock()
	defer net.nodesMtx.Unlock()
	if net.leader == nil {
		return nil, ErrNoLeader
	}
	return net.leader.node.getHistory(ctx, scripthash)
}

func (net *Network) GetListUnspent(ctx context.Context, scripthash string) (ListUnspentResult, error) {
	if !net.started {
		return nil, ErrNoNetwork
	}
	net.nodesMtx.Lock()
	defer net.nodesMtx.Unlock()
	if net.leader == nil {
		return nil, ErrNoLeader
	}
	return net.leader.node.getListUnspent(ctx, scripthash)
}

func (net *Network) GetTransaction(ctx context.Context, txid string) (*GetTransactionResult, error) {
	if !net.started {
		return nil, ErrNoNetwork
	}
	net.nodesMtx.Lock()
	defer net.nodesMtx.Unlock()
	if net.leader == nil {
		return nil, ErrNoLeader
	}
	if net.leader == nil {
		return nil, ErrNoLeader
	}
	return net.leader.node.getTransaction(ctx, txid)
}

func (net *Network) GetRawTransaction(ctx context.Context, txid string) (string, error) {
	if !net.started {
		return "", ErrNoNetwork
	}
	net.nodesMtx.Lock()
	defer net.nodesMtx.Unlock()
	if net.leader == nil {
		return "", ErrNoLeader
	}
	return net.leader.node.getRawTransaction(ctx, txid)
}

func (net *Network) Broadcast(ctx context.Context, rawTx string) (string, error) {
	if !net.started {
		return "", ErrNoNetwork
	}
	net.nodesMtx.Lock()
	defer net.nodesMtx.Unlock()
	if net.leader == nil {
		return "", ErrNoLeader
	}
	return net.leader.node.broadcast(ctx, rawTx)
}

func (net *Network) EstimateFeeRate(ctx context.Context, confTarget int64) (int64, error) {
	if !net.started {
		return 0, ErrNoNetwork
	}
	net.nodesMtx.Lock()
	defer net.nodesMtx.Unlock()
	if net.leader == nil {
		return 0, ErrNoLeader
	}
	return net.leader.node.estimateFeeRate(ctx, confTarget)
}
