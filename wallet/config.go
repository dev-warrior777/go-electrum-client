package wallet

import (
	"net"
	"net/url"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"golang.org/x/net/proxy"
)

const (
	appName = "goele"
)

type Config struct {
	// The blockchain, Bitcoin, Dash, etc
	Chain CoinType

	// Network parameters. Set mainnet, testnet using this.
	Params *chaincfg.Params

	// Store the seed in encrypted storage
	StoreEncSeed bool

	// The user-agent that shall be visible to the network
	UserAgent string

	// Location of the data directory
	DataDir string

	// An implementation of the Datastore interface
	DB Datastore

	// If you wish to connect to a single trusted electrumX peer set this. TODO:
	TrustedPeer net.Addr

	// A Tor proxy can be set here causing the wallet will use Tor. TODO:
	Proxy proxy.Dialer

	// The default fee-per-byte for each level
	LowFee    uint64
	MediumFee uint64
	HighFee   uint64

	// The highest allowable fee-per-byte
	MaxFee uint64

	// External API to query to look up fees. If this field is nil then the default fees will be used.
	// If the API is unreachable then the default fees will likewise be used. If the API returns a fee
	// greater than MaxFee then the MaxFee will be used in place. The API response must be formatted as
	// { "fastestFee": 40, "halfHourFee": 20, "hourFee": 10 }
	FeeAPI url.URL

	// Disable the exchange rate provider
	DisableExchangeRates bool
}

func NewDefaultConfig() *Config {
	return &Config{
		Chain:                Bitcoin,
		Params:               &chaincfg.MainNetParams,
		UserAgent:            appName,
		DataDir:              btcutil.AppDataDir(appName, false),
		DB:                   nil, // TODO: update for concrete impl
		DisableExchangeRates: true,
	}
}
