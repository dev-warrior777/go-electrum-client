package wallet

import (
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
)

const (
	appName = "goele"
)

type Config struct {
	// The blockchain, Bitcoin, Dash, etc
	Chain CoinType

	// Network parameters. Set mainnet, testnet using this.
	Params *chaincfg.Params

	// The user-agent that shall be visible to peers
	UserAgent string

	// Location of the data directory
	DataDir string

	// An implementation of the Datastore interface
	DB Datastore
}

func NewDefaultConfig() *Config {
	return &Config{
		Chain:     Bitcoin,
		Params:    &chaincfg.MainNetParams,
		UserAgent: appName,
		DataDir:   btcutil.AppDataDir(appName, false),
		DB:        nil, // TODO: update for concrete impl
	}
}
