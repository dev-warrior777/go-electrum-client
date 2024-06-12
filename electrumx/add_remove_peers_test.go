package electrumx

import (
	"sync"
	"testing"
)

var n = &Network{
	config:   &ElectrumXConfig{},
	started:  false,
	startMtx: sync.Mutex{},
	nodes: []*MultiNode{
		{
			id:      uint32(0),
			trusted: true,
			node:    &Node{state: CONNECTED},
		},
	},
	leader: &MultiNode{
		id:      uint32(0),
		trusted: true,
		node:    &Node{state: CONNECTED},
	},
	nodesMtx: sync.Mutex{},
}

func TestAddPeers(t *testing.T) {
	if len(n.nodes) != 1 {
		t.Fatal("incorrect nodes length")
	}
	n.addPeer(&MultiNode{
		id:      1,
		trusted: false,
		node: &Node{
			state: CONNECTED,
		},
	})
	if len(n.nodes) != 2 {
		t.Fatal("incorrect nodes length after add")
	}
	n.addPeer(&MultiNode{
		id:      2,
		trusted: false,
		node: &Node{
			state: CONNECTED,
		},
	})
	if len(n.nodes) != 3 {
		t.Fatal("incorrect nodes length after add")
	}
	// remove #id=1
	err := n.removePeer(&MultiNode{
		id:      1,
		trusted: false,
		node: &Node{
			state: CONNECTED,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(n.nodes) != 2 {
		t.Fatal("incorrect nodes length after add")
	}
	// remove leader id=0
	err = n.removePeer(&MultiNode{
		id:      0,
		trusted: false,
		node: &Node{
			state: CONNECTED,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(n.nodes) != 1 {
		t.Fatal("incorrect nodes length after add")
	}
	if n.leader.id != 2 {
		t.Fatal("new leader not elected")
	}
	// remove id=2
	err = n.removePeer(&MultiNode{
		id:      2,
		trusted: false,
		node: &Node{
			state: CONNECTED,
		},
	})
	if err == nil {
		t.Fatal("expected error cannot remove")
	}
	if len(n.nodes) != 1 {
		t.Fatal("incorrect nodes length after add")
	}
}
