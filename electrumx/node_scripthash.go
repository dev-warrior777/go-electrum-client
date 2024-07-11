package electrumx

import (
	"context"
	"errors"
)

func (n *Node) scriptHashNotify(nodeCtx context.Context) error {
	scriptHashNotifyChan := n.getScripthashNotify()
	if scriptHashNotifyChan == nil {
		return errors.New("server scripthash notify channel is nil")
	}

	// TODO: make a queue & use 'range over incoming channel' pattern

	go func() {
		for {
			select {
			case <-nodeCtx.Done():
				<-n.server.conn.Done()
				return
			case scriptHashStatusResult, ok := <-scriptHashNotifyChan:
				if !ok {
					return
				}
				// forward to client wallet_synchronize.go - can block
				n.clientScriptHashNotify <- scriptHashStatusResult
			}
		}
	}()

	return nil
}
