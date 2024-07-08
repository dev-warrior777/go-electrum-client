package electrumx

import (
	"context"
	"errors"
	"fmt"
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
				fmt.Printf("nodeCtx.Done - in scriptHashNotify %s - exiting thread\n", n.serverAddr)
				return
			case scriptHashStatusResult, ok := <-scriptHashNotifyChan:
				if !ok {
					fmt.Printf("scripthash notify channel was closed %s - exiting thread\n", n.serverAddr)
					return
				}
				// forward to client wallet_synchronize.go - can block
				n.clientScriptHashNotify <- scriptHashStatusResult
			}
		}
	}()

	return nil
}
