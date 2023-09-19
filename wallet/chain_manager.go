package wallet

type ChainManager struct {
	Chain CoinType
	Net   string
	//TODO:
}

func NewChainManager(cfg *Config) *ChainManager {
	var chainNet string
	switch cfg.Params.Name {
	case "testnet", "testnet3":
		chainNet = "testnet"
	case "regtest", "simnet":
		chainNet = "simnet"
	default:
		chainNet = "mainnet"
	}

	return &ChainManager{
		Chain: cfg.Chain,
		Net:   chainNet,
	}
}
