package electrumx

import (
	"context"
	"errors"
	"fmt"
)

func (n *Node) scriptHashNotify(nodeCtx context.Context) error {
	// get a channel to receive scripthash notifications from this node's <- server connection
	scriptHashNotifyChan := n.getScripthashNotify()
	if scriptHashNotifyChan == nil {
		return errors.New("server scripthash notify channel is nil")
	}

	// start scripthash queue with depth 8
	qchan := make(chan *ScripthashStatusResult, 8)
	go n.scriptHashQueue(nodeCtx, qchan)

	go func() {
		defer close(qchan)
		fmt.Println("=== Waiting for Scripthash Notifications")
		for {
			if nodeCtx.Err() != nil {
				<-n.server.conn.done
				return
			}
			// from server into queue
			ntfn := <-scriptHashNotifyChan
			qchan <- ntfn
		}
	}()

	return nil
}

func (n *Node) scriptHashQueue(nodeCtx context.Context, qchan <-chan *ScripthashStatusResult) {
	for {
		if nodeCtx.Err() != nil {
			<-n.server.conn.Done()
			return
		}

		for ntfn := range qchan {
			if ntfn == nil {
				return
			}
			n.clientScriptHashNotify <- ntfn
		}
	}
}
