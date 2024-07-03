package client

import (
	"net/url"
	"os"
	"path/filepath"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"golang.org/x/net/proxy"

	"github.com/dev-warrior777/go-electrum-client/electrumx"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

const (
	appName      = "goele"
	DbTypeBolt   = "bbolt"
	DbTypeSqlite = "sqlite"
)

type ClientConfig struct {
	// The blockchain, btc, dash, etc
	Chain wallet.CoinType

	// Size of a block header for this chain - defaults to 80 bytes.
	BlockHeaderSize int

	// Network parameters.
	Params *chaincfg.Params

	// Store the seed in encrypted storage
	StoreEncSeed bool

	// The user-agent visible to the network
	UserAgent string

	// Location of the data directory
	DataDir string

	// Database implementation type (bbolt or sqlite)
	DbType string

	// An implementation of the Datastore interface
	DB wallet.Datastore

	// Currently we use this electrumX server to bootstrap others so it should
	// be set.
	TrustedPeer *electrumx.NodeServerAddr

	// A Tor proxy can be set here. TODO:
	Proxy proxy.Dialer

	// The default fee-per-byte for each level
	LowFee    int64
	MediumFee int64
	HighFee   int64

	// The highest allowable fee-per-byte
	MaxFee int64

	// External API to query to look up fees. If this field is nil then the default fees will be used.
	// If the API is unreachable then the default fees will likewise be used. If the API returns a fee
	// greater than MaxFee then the MaxFee will be used instead.
	FeeAPI url.URL

	// Disable the exchange rate provider
	DisableExchangeRates bool

	// If not testing do not overwrite existing wallet files
	Testing bool

	// Test RPC server
	RPCTestPort int
}

func NewDefaultConfig() *ClientConfig {
	return &ClientConfig{
		Chain:                wallet.Bitcoin,
		Params:               &chaincfg.MainNetParams,
		BlockHeaderSize:      80,
		UserAgent:            appName,
		DataDir:              btcutil.AppDataDir(appName, false),
		DbType:               DbTypeBolt,
		DB:                   nil, // concrete impl
		DisableExchangeRates: true,
		RPCTestPort:          8888,
	}
}
func (cc *ClientConfig) MakeWalletConfig() *wallet.WalletConfig {
	wc := wallet.WalletConfig{
		Chain:        cc.Chain,
		Params:       cc.Params,
		StoreEncSeed: cc.StoreEncSeed,
		DataDir:      cc.DataDir,
		DbType:       cc.DbType,
		DB:           cc.DB,
		LowFee:       cc.LowFee,
		MediumFee:    cc.MediumFee,
		HighFee:      cc.HighFee,
		MaxFee:       cc.MaxFee,
		Testing:      cc.Testing,
	}
	return &wc
}

func (cc *ClientConfig) MakeElectrumXConfig() *electrumx.ElectrumXConfig {
	nc := electrumx.ElectrumXConfig{
		Chain:           cc.Chain,
		Params:          cc.Params,
		BlockHeaderSize: cc.BlockHeaderSize,
		UserAgent:       cc.UserAgent,
		DataDir:         cc.DataDir,
		TrustedPeer:     cc.TrustedPeer,
		Proxy:           cc.Proxy,
		Testing:         cc.Testing,
	}
	return &nc
}

func GetConfigPath() (string, error) {
	userCfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appPath := filepath.Join(userCfgDir, appName)
	err = os.MkdirAll(appPath, os.ModeDir|0777)
	if err != nil {
		return "", err
	}
	return appPath, nil
}
