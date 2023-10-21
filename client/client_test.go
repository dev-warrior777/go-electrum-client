package client

import (
	"fmt"
	"testing"

	"github.com/dev-warrior777/go-electrum-client/electrumx"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

type GenElectrumClient struct {
	c *Config
	w wallet.ElectrumWallet
	n electrumx.ElectrumXNode
}

func NewGenElectrumClient() ElectrumClient {
	ec := GenElectrumClient{
		c: NewDefaultConfig(),
		w: nil,
		n: nil,
	}
	return &ec
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

func (gec GenElectrumClient) Config() *Config {
	return gec.c
}
func (gec *GenElectrumClient) Wallet() wallet.ElectrumWallet {
	return gec.w
}
func (gec *GenElectrumClient) Node() electrumx.ElectrumXNode {
	return gec.n
}

func TestMakeGenClient(t *testing.T) {
	gex := NewGenElectrumClient()
	fmt.Println(gex.Config().DataDir)
}
