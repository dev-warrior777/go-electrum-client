package btc

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/dev-warrior777/go-electrum-client/client"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

func init() {
	cfg := client.NewDefaultConfig()
	cfg.Chain = wallet.Bitcoin
	cfg.StoreEncSeed = true
	cfg.Testing = true
	cfg.Params = &chaincfg.RegressionNetParams

	NewBtcElectrumClient(cfg).RPCServe()
}

func TestRpc(t *testing.T) {
}
