package chain

type ChainType string
type NetType string

const (
	NetTypeMainnet = "mainnet"
	NetTypeTestnet = "testnet"

	ChainBTC ChainType = "BTC"
	//TODO:
)

type ChainManager struct {
	Chain ChainType
	Net   NetType
	//TODO:
}

func NewChainManager(net NetType, chain ChainType) *ChainManager {
	return &ChainManager{
		Chain: chain,
		Net:   net,
	}
}
