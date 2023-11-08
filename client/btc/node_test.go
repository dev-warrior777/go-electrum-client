package btc

import (
	"fmt"
	"testing"

	"github.com/dev-warrior777/go-electrum-client/client"
)

func TestNodeCreate(t *testing.T) {
	c := NewBtcElectrumClient(client.NewDefaultConfig())
	fmt.Println(c.GetConfig().DataDir)
	c.CreateNode(client.SingleNode)
	n := c.GetNode()
	fmt.Println(n)
}

func TestMultiNodeCreate(t *testing.T) {
	fmt.Println("TBD:")
}
