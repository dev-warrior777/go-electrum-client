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

	go func() {
		fmt.Println("leader node waiting for scripthash notifications")
		for {
			select {
			case <-nodeCtx.Done():
				n.state = DISCONNECTING
				fmt.Println("nodeCtx.Done - in scriptHashNotify - exiting thread")
				n.server.conn.cancel()
				<-n.server.conn.Done()
				return
			case scriptHashStatusResult, ok := <-scriptHashNotifyChan:
				if !ok {
					n.state = DISCONNECTING
					fmt.Println("scripthash notify channel was closed - exiting thread")
					return
				}
				// forward to client wallet_synchronize.go - can block
				n.rcvScriptHashNotify <- scriptHashStatusResult
			}
		}
	}()

	return nil
}
