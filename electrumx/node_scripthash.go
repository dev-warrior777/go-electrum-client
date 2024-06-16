package electrumx

import (
	"context"
	"errors"
	"fmt"
)

func (n *Node) scriptHashNotify(ctx context.Context) error {
	scriptHashNotifyChan := n.getScripthashNotify()
	if scriptHashNotifyChan == nil {
		return errors.New("server scripthash notify channel is nil")
	}

	go func() {
		fmt.Println("leader node waiting for scripthash notifications")
		for {
			select {
			case <-ctx.Done():
				n.state = DISCONNECTING
				fmt.Println("ctx.Done - in leader node scriptHashNotify - exiting thread")
				return
			case scriptHashStatusResult, ok := <-scriptHashNotifyChan:
				if !ok {
					n.state = DISCONNECTING
					fmt.Println("leader node scripthash notify channel was closed - exiting thread")
					return
				}
				// forward to client wallet_synchronize.go - can block
				n.rcvScriptHashNotify <- scriptHashStatusResult
			}
		}
	}()

	return nil
}
