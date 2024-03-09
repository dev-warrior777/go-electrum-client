package main

// Run goele as an app for testing

import (
	"errors"
	"flag"
	"fmt"
	"os"
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
			// Net: "ssl", Addr: "testnet.aranguren.org:51002",
			// Net: "tcp", Addr: "testnet.aranguren.org:51001",
			Net: "ssl", Addr: "blockstream.info:993",
			// Net: "tcp", Addr: "blockstream.info:143",
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

	// make basic client
	ec := btc.NewBtcElectrumClient(cfg)

	// start client, create node & sync headers
	err = ec.Start()
	if err != nil {
		ec.Stop()
		fmt.Printf("%v - exiting.\n%s\n", err, checkSimnetHelp(cfg))
		os.Exit(1)
	}

	// make the client's wallet

	if net == "regtest" {

		// for non-mainnet testing recreate a wallet with a known set of keys ..
		// var mnemonic = "jungle pair grass super coral bubble tomato sheriff pulp cancel luggage wagon"
		// err := ec.RecreateWallet("abc", mnemonic)
		//
		// ...therafter load the wallet; careful no params change in harness script
		// that invalidate current wallet txs.
		//
		err := ec.LoadWallet("abc")
		if err != nil {
			ec.GetNode().Stop()
			fmt.Println(err, " - exiting")
			os.Exit(1)
		}

	} else if net == "testnet3" {

		// err := ec.RecreateWallet("abc", "canyon trip truly ritual lonely quiz romance rose alone journey like bronze")
		// if err != nil {
		// 	ec.GetNode().Stop()
		// 	fmt.Println(err, " - exiting")
		// 	os.Exit(1)
		// }

		// Or

		// uncomment to create new ->
		// err := ec.CreateWallet("abc")
		// if err != nil {
		// 	ec.GetNode().Stop()
		// 	fmt.Println(err, " - exiting")
		// 	os.Exit(1)
		// }
		//<- uncomment to create

		// comment out when doing initial creation_->
		// load the client's wallet
		// canyon trip truly ritual lonely quiz romance rose alone journey like bronze
		err := ec.LoadWallet("abc")
		if err != nil {
			ec.GetNode().Stop()
			fmt.Println(err, " - exiting")
			os.Exit(1)
		}
		//<- comment out when doing initial creation

	} else if net == "mainnet" {
		// production usage: load the client's wallet
		err := ec.LoadWallet(pass)
		if err != nil {
			ec.GetNode().Stop()
			fmt.Println(err, " - exiting")
			os.Exit(1)
		}
	}

	// set up Notify for all our already given out receive addresses (getunusedaddress)
	// and broadcasted change addresses in order to receive any changes to the state of
	// the address history from the node
	err = ec.SyncWallet()
	if err != nil {
		ec.Stop()
		fmt.Println(err, " - exiting")
		os.Exit(1)
	}

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
