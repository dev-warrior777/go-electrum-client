package client

import (
	"fmt"
	"testing"

	"github.com/dev-warrior777/go-electrum-client/electrumx"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

type GenElectrumClient struct {
	c *ClientConfig
	w wallet.ElectrumWallet
	n electrumx.ElectrumXNode
}

type GenElectrumNode struct {
	_ *electrumx.NodeConfig
	_ *electrumx.ElectrumXSvrConn
}

func NewGenElectrumClient() ElectrumClient {
	ec := GenElectrumClient{
		c: NewDefaultConfig(),
		w: nil,
		n: nil,
	}
	return &ec
}

func (gec *GenElectrumClient) GetConfig() *ClientConfig {
	return gec.c
}
func (gec *GenElectrumClient) GetWallet() wallet.ElectrumWallet {
	return gec.w
}
func (gec *GenElectrumClient) GetNode() electrumx.ElectrumXNode {
	return gec.n
}

func (gec *GenElectrumClient) CreateWallet(pw string) error {
	return nil
}
func (gec *GenElectrumClient) RecreateElectrumWallet(pw, mnenomic string) error {
	return nil
}
func (gec *GenElectrumClient) LoadWallet(pw string) error {
	return nil
}

func (gec *GenElectrumClient) CreateNode() {
	gec.n = nil
}

func (gec *GenElectrumClient) SyncClientHeaders() error {
	return nil
}

func (gec *GenElectrumClient) SubscribeClientHeaders() error {
	return nil
}

func TestMakeGenClient(t *testing.T) {
	gex := NewGenElectrumClient()
	fmt.Println(gex.GetConfig().DataDir)
}
