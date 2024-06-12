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

func (n *Network) GetTipChangeNotify() chan int64 {
	return n.rcvTipChangeNotify
}

func (n *Network) GetScripthashNotify() chan *ScripthashStatusResult {
	return n.rcvScripthashNotify
}

func (n *Network) Start(ctx context.Context) error {
	if n.config.TrustedPeer == nil {
		return errors.New("a trusted peer is required in config")
	}
	n.startMtx.Lock()
	defer n.startMtx.Unlock()
	if n.started {
		return errors.New("network already started")
	}
	return n.start(ctx)
}

func (n *Network) start(ctx context.Context) error {
	// start from our trusted node as leader
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	node, err := newNode(n.config.TrustedPeer, true, n.headers, n.rcvTipChangeNotify, n.rcvScripthashNotify)
	if err != nil {
		return err
	}
	network := n.config.Chain.String()
	nettype := n.config.Params.Name
	genesis := n.config.Params.GenesisHash.String()
	err = node.start(ctx, network, nettype, genesis)
	if err != nil {
		return err
	}
	m := newMultiNodeWithId(true, node)
	n.addPeer(m)
	n.leader = m

	// TODO: ...bootstrap peers loop with leader

	n.started = true
	return nil
}

func (n *Network) addPeer(m *MultiNode) {
	n.nodes = append(n.nodes, m)
}

func (n *Network) removePeer(m *MultiNode) error {
	nodesLen := len(n.nodes)
	if nodesLen < 2 {
		return fmt.Errorf("cannot remove peer node - ony have %d node(s) in peer list", nodesLen)
	}
	currNodes := n.nodes
	newNodes := make([]*MultiNode, 0, nodesLen-1)
	for _, multi := range currNodes {
		if m.id != multi.id {
			newNodes = append(newNodes, multi)
		}
	}
	n.nodes = newNodes
	if n.leader.id == m.id {
		// make a new random leader - there are 1 or more nodes in the slice
		n.leader = newNodes[0]
	}
	return nil
}

// -----------------------------------------------------------------------------
// Nodes monitor
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
// API
// -----------------------------------------------------------------------------

// func (n *Network) SubscribeHeaders(ctx context.Context) (*HeadersNotifyResult, error) {
// 	if !n.started {
// 		return nil, ErrNoNetwork
// 	}
// 	n.nodesMtx.Lock()
// 	defer n.nodesMtx.Unlock()
// 	if n.leader == nil {
// 		return nil, ErrNoLeader
// 	}
// 	n.leader.node.SubscribeHeaders(ctx)
// }

func (n *Network) SubscribeScripthashNotify(ctx context.Context, scripthash string) (*ScripthashStatusResult, error) {
	if !n.started {
		return nil, ErrNoNetwork
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	if n.leader == nil {
		return nil, ErrNoLeader
	}
	return n.leader.node.subscribeScripthashNotify(ctx, scripthash)
}

func (n *Network) UnsubscribeScripthashNotify(ctx context.Context, scripthash string) {
	if !n.started {
		return
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	n.leader.node.unsubscribeScripthashNotify(ctx, scripthash)
}

func (n *Network) BlockHeader(height int64) (*wire.BlockHeader, error) {
	if !n.started {
		return nil, ErrNoNetwork
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	if n.leader == nil {
		return nil, ErrNoLeader
	}
	return n.leader.node.getBlockHeader(height), nil
}

func (n *Network) BlockHeaders(startHeight int64, blockCount int64) ([]*wire.BlockHeader, error) {
	if !n.started {
		return nil, ErrNoNetwork
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	if n.leader == nil {
		return nil, ErrNoLeader
	}
	return n.leader.node.getBlockHeaders(startHeight, blockCount)
}

func (n *Network) GetHistory(ctx context.Context, scripthash string) (HistoryResult, error) {
	if !n.started {
		return nil, ErrNoNetwork
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	if n.leader == nil {
		return nil, ErrNoLeader
	}
	return n.leader.node.getHistory(ctx, scripthash)
}

func (n *Network) GetListUnspent(ctx context.Context, scripthash string) (ListUnspentResult, error) {
	if !n.started {
		return nil, ErrNoNetwork
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	if n.leader == nil {
		return nil, ErrNoLeader
	}
	return n.leader.node.getListUnspent(ctx, scripthash)
}

func (n *Network) GetTransaction(ctx context.Context, txid string) (*GetTransactionResult, error) {
	if !n.started {
		return nil, ErrNoNetwork
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	if n.leader == nil {
		return nil, ErrNoLeader
	}
	if n.leader == nil {
		return nil, ErrNoLeader
	}
	return n.leader.node.getTransaction(ctx, txid)
}

func (n *Network) GetRawTransaction(ctx context.Context, txid string) (string, error) {
	if !n.started {
		return "", ErrNoNetwork
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	if n.leader == nil {
		return "", ErrNoLeader
	}
	return n.leader.node.getRawTransaction(ctx, txid)
}

func (n *Network) Broadcast(ctx context.Context, rawTx string) (string, error) {
	if !n.started {
		return "", ErrNoNetwork
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	if n.leader == nil {
		return "", ErrNoLeader
	}
	return n.leader.node.broadcast(ctx, rawTx)
}

func (n *Network) EstimateFeeRate(ctx context.Context, confTarget int64) (int64, error) {
	if !n.started {
		return 0, ErrNoNetwork
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	if n.leader == nil {
		return 0, ErrNoLeader
	}
	return n.leader.node.estimateFeeRate(ctx, confTarget)
}
