package electrumx

type Network string

var Regtest Network = "regtest"
var Testnet Network = "testnet"
var Mainnet Network = "mainnet"

var DebugMode bool

type ElectrumXNode interface {
}
