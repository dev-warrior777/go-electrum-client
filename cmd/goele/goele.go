package main

// Run goele as an app for testing

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/dev-warrior777/go-electrum-client/client"
	"github.com/dev-warrior777/go-electrum-client/client/btc"
	"github.com/dev-warrior777/go-electrum-client/electrumx"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

var (
	coins = []string{"btc"} // add as implemented
	nets  = []string{"mainnet", "testnet", "regtest", "simnet"}
)

func makeBasicConfig(coin, net string) (*client.ClientConfig, error) {
	contains := func(s []string, str string) bool {
		for _, v := range s {
			if v == str {
				return true
			}
		}
		return false
	}
	if !contains(coins, coin) {
		return nil, errors.New("invalid coin")
	}
	if !contains(nets, net) {
		return nil, errors.New("invalid net")
	}
	switch coin {
	case "btc":
	default:
		return nil, errors.New("invalid coin")
	}
	cfg := client.NewDefaultConfig()
	cfg.Chain = wallet.Bitcoin
	cfg.StoreEncSeed = true
	appDir, err := client.GetConfigPath()
	if err != nil {
		return nil, err
	}
	coinNetDir := filepath.Join(appDir, coin, net)
	err = os.MkdirAll(coinNetDir, os.ModeDir|0777)
	if err != nil {
		return nil, err
	}
	cfg.DataDir = coinNetDir
	switch net {
	case "regtest", "simnet":
		cfg.Params = &chaincfg.RegressionNetParams
		cfg.TrustedPeer = electrumx.ServerAddr{
			// Net: "ssl", Addr: "127.0.0.1:57002", // debug server
			Net: "ssl", Addr: "127.0.0.1:53002",
		}
		cfg.StoreEncSeed = true
		cfg.Testing = true
	case "testnet":
		cfg.Params = &chaincfg.TestNet3Params
		cfg.TrustedPeer = electrumx.ServerAddr{
			// Net: "ssl", Addr: "testnet.aranguren.org:51002", // down?
			// Net: "ssl", Addr: "testnet.hsmiths.com:53012", // down?
			// Net: "ssl", Addr: "electrum.blockstream.info:60002", // no verbose gtx
			// Net: "ssl", Addr: "blackie.c3-soft.com:57006", // down?
			// Net: "tcp", Addr: "blackie.c3-soft.com:57005", // down
			// Net: "ssl", Addr: "testnet.qtornado.com:51002",
			Net: "tcp", Addr: "testnet.qtornado.com:51001",
		}
		cfg.StoreEncSeed = true
		cfg.Testing = true
	case "mainnet":
		cfg.Params = &chaincfg.MainNetParams
		cfg.TrustedPeer = electrumx.ServerAddr{
			Net: "ssl", Addr: "elx.bitske.com:50002",
		}
		cfg.StoreEncSeed = false
		cfg.Testing = false
	}
	return cfg, nil
}

func configure() (string, *client.ClientConfig, error) {
	coin := flag.String("coin", "btc", "coin name")
	net := flag.String("net", "regtest", "network type; testnet, mainnet, regtest")
	pass := flag.String("pass", "", "wallet password")
	flag.Parse()
	fmt.Println("coin:", *coin)
	fmt.Println("net:", *net)
	cfg, err := makeBasicConfig(*coin, *net)
	return *pass, cfg, err
}

func checkSimnetHelp(cfg *client.ClientConfig) string {
	var help string
	switch cfg.Params {
	case &chaincfg.RegressionNetParams:
		help = "check out simnet harness scripts at client/btc/test_harness\n" +
			"README.md, src_harness.sh & ex.sh\n" +
			"Then when goele starts navigate to client/btc/rpctest and use the\n" +
			"minimalist rpc test client"
	default:
		help = "is ElectrumX server up and running?"
	}
	return help
}

func main() {
	fmt.Println("Goele", client.GoeleVersion)
	pass, cfg, err := configure()
	if err != nil {
		fmt.Println(err, " - exiting")
		os.Exit(1)
	}
	net := cfg.Params.Name
	fmt.Println(net)

	cfg.DbType = "sqlite"
	fmt.Println(cfg.DbType)

	// make basic client
	ec := btc.NewBtcElectrumClient(cfg)

	// start client, create node & sync headers
	clientCtx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	err = ec.Start(clientCtx)
	if err != nil {
		ec.Stop()
		fmt.Printf("%v - exiting.\n%s\n", err, checkSimnetHelp(cfg))
		os.Exit(1)
	}

	feeRate, _ := ec.FeeRate(clientCtx, 6)
	fmt.Println(feeRate)

	// to make the client's wallet:
	// - for regtest/testnet testing recreate a wallet with a known set of keys.
	// - use the mkwallet and rmwallet tools to create, recreate a wallet at the
	//   configured location
	// - the rmwallet tool removes a wallet from the configured location.
	//   regtest & testnet only

	switch net {
	case "regtest":
		// mnemonic := "jungle pair grass super coral bubble tomato sheriff pulp cancel luggage wagon"
		// err := ec.RecreateWallet("abc", mnemonic)
		err := ec.LoadWallet("abc")
		if err != nil {
			ec.Stop()
			fmt.Println(err, " - exiting")
			os.Exit(1)
		}
	case "testnet3":
		// mnemonic := "canyon trip truly ritual lonely quiz romance rose alone journey like bronze"
		// err := ec.RecreateWallet("abc", mnemonic)
		ec.LoadWallet("abc")
		if err != nil {
			ec.GetNode().Stop()
			fmt.Println(err, " - exiting")
			os.Exit(1)
		}
	case "mainnet":
		// production usage: load the client's wallet
		err := ec.LoadWallet(pass)
		if err != nil {
			ec.Stop()
			fmt.Println(err, " - exiting")
			os.Exit(1)
		}
	default:
		ec.Stop()
		fmt.Printf("unknown net %s - exiting\n", net)
		os.Exit(1)
	}

	// Set up Notify for all our already given out receive addresses (getunusedaddress)
	// and broadcasted change addresses in order to receive any changes to the state of
	// the address history back from the node
	err = ec.SyncWallet(clientCtx)
	if err != nil {
		ec.Stop()
		fmt.Println(err, " - exiting")
		os.Exit(1)
	}

	// // bcrt1qmurgx56cv82p4u3jqzf9ndwnsyw5gudguh77wp
	// // tb1qqf85rm98kcls6s83kyumefdqaajkyuj0nun7wu
	// start := time.Now().Unix()
	// for i := 0; i < 10000; i++ {
	// 	fmt.Printf("%d\n", i)
	// 	var addr string
	// 	switch net {
	// 	case "regtest":
	// 		addr = "bcrt1qmurgx56cv82p4u3jqzf9ndwnsyw5gudguh77wp"
	// 	case "testnet, testnet3":
	// 		addr = "tb1qqf85rm98kcls6s83kyumefdqaajkyuj0nun7wu"
	// 	case "mainnet":
	// 		addr = ""
	// 	}
	// 	h, err := ec.GetAddressHistory(clientCtx, addr)
	// 	if err != nil {
	// 		fmt.Printf("\n")
	// 	}
	// 	for i, hist := range h {
	// 		fmt.Printf("%d %v %d %s\n", i, hist.Height, hist.Fee, hist.TxHash)
	// 	}
	// }
	// end := time.Now().Unix()
	// fmt.Printf("elapsed %ds\n", end-start)

	// for testing only
	err = btc.RPCServe(ec)
	if err != nil {
		ec.Stop()
		fmt.Println(err, " - exiting")
		os.Exit(1)
	}

	// SIGINT kills the node server(s) & test rpc server

	ec.Stop()
}
