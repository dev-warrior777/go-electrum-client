package electrumx

// Implements failover for ElectrumX servers

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

var ErrNoNetwork = errors.New("network not started")

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

func (m *MultiNode) getId() uint32 {
	return m.id
}

type Network struct {
	config   *ElectrumXConfig
	started  bool
	startMtx sync.Mutex
	nodes    []*MultiNode
	leader   *MultiNode
	nodesMtx sync.Mutex
}

func NewNetwork(config *ElectrumXConfig) *Network {
	n := &Network{
		config:  config,
		started: false,
		nodes:   make([]*MultiNode, 0),
		leader:  nil, // explicit for readability
	}
	return n
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
	// start from our trusted node
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	node, err := newNode(n.config.TrustedPeer)
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
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	n.nodes = append(n.nodes, m)
}

func (n *Network) removePeer(m *MultiNode) error {
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
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

// TODO: remove when remove single-node
func (n *Network) GetHeadersNotify() (<-chan *HeadersNotifyResult, error) {
	if !n.started {
		return nil, ErrNoNetwork
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	// return x.network.HeadersNotify, nil
	return nil, nil
}

func (n *Network) SubscribeHeaders(ctx context.Context) (*HeadersNotifyResult, error) {
	if !n.started {
		return nil, ErrNoNetwork
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	// return x.server.Conn.SubscribeHeaders(ctx)
	return nil, nil
}

func (n *Network) GetScripthashNotify() (<-chan *ScripthashStatusResult, error) {
	if !n.started {
		return nil, ErrNoNetwork
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	// return x.scripthashNotify, nil
	return nil, nil
}

func (n *Network) SubscribeScripthashNotify(ctx context.Context, scripthash string) (*ScripthashStatusResult, error) {
	if !n.started {
		return nil, ErrNoNetwork
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	// return x.server.Conn.SubscribeScripthash(ctx, scripthash)
	return nil, nil
}

func (n *Network) UnsubscribeScripthashNotify(ctx context.Context, scripthash string) {
	if !n.started {
		return
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	// x.server.Conn.UnsubscribeScripthash(ctx, scripthash)
}

func (n *Network) BlockHeader(ctx context.Context, height int64) (string, error) {
	if !n.started {
		return "", ErrNoNetwork
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	// return x.server.Conn.BlockHeader(ctx, uint32(height))
	return "", nil
}

func (n *Network) BlockHeaders(ctx context.Context, startHeight int64, blockCount int) (*GetBlockHeadersResult, error) {
	if !n.started {
		return nil, ErrNoNetwork
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	// return x.server.Conn.BlockHeaders(ctx, startHeight, blockCount)
	return nil, nil
}

func (n *Network) GetHistory(ctx context.Context, scripthash string) (HistoryResult, error) {
	if !n.started {
		return nil, ErrNoNetwork
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	// return x.server.Conn.GetHistory(ctx, scripthash)
	return nil, nil
}

func (n *Network) GetListUnspent(ctx context.Context, scripthash string) (ListUnspentResult, error) {
	if !n.started {
		return nil, ErrNoNetwork
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	// return x.server.Conn.GetListUnspent(ctx, scripthash)
	return nil, nil
}

func (n *Network) GetTransaction(ctx context.Context, txid string) (*GetTransactionResult, error) {
	if !n.started {
		return nil, ErrNoNetwork
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	// return x.server.Conn.GetTransaction(ctx, txid)
	return nil, nil
}

func (n *Network) GetRawTransaction(ctx context.Context, txid string) (string, error) {
	if !n.started {
		return "", ErrNoNetwork
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	// return x.server.Conn.GetRawTransaction(ctx, txid)
	return "", nil
}

func (n *Network) Broadcast(ctx context.Context, rawTx string) (string, error) {
	if !n.started {
		return "", ErrNoNetwork
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	// return x.server.Conn.Broadcast(ctx, rawTx)
	return "", nil
}

func (n *Network) EstimateFeeRate(ctx context.Context, confTarget int64) (int64, error) {
	if !n.started {
		return 0, ErrNoNetwork
	}
	n.nodesMtx.Lock()
	defer n.nodesMtx.Unlock()
	// return x.server.Conn.EstimateFee(ctx, confTarget)
	return 0, nil
}
