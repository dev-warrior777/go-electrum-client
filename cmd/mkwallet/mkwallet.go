package main

// Run create or recreate a wallet for testing

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/dev-warrior777/go-electrum-client/client"
	"github.com/dev-warrior777/go-electrum-client/client/btc"
	"github.com/dev-warrior777/go-electrum-client/electrumx"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

var (
	coins = []string{"btc"} // add as implemented
	nets  = []string{"mainnet", "testnet", "testnet3", "regtest", "simnet"}
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
	case "testnet", "testnet3":
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

func configure() (string, string, string, *client.ClientConfig, error) {
	coin := flag.String("coin", "btc", "coin name")
	net := flag.String("net", "regtest", "network type; testnet, mainnet, regtest")
	pass := flag.String("pass", "", "wallet password")
	action := flag.String("action", "create", "action: 'create'a new wallet or 'recreate' from seed")
	seed := flag.String("seed", "", "'seed words for recreate' inside ''; example: 'word1 word2 ... word12'")
	test_wallet := flag.Bool("tw", false, "known test wallets override for regtest/testnet")
	flag.Parse()
	fmt.Println("coin:", *coin)
	fmt.Println("net:", *net)
	fmt.Println("action:", *action)
	fmt.Println("pass:", *pass)
	fmt.Println("seed:", *seed)
	if *test_wallet {
		switch *net {
		case "regtest", "simnet":
			*seed = "jungle pair grass super coral bubble tomato sheriff pulp cancel luggage wagon"
		case "testnet", "testnet3":
			*seed = "canyon trip truly ritual lonely quiz romance rose alone journey like bronze"
		default:
			return "", "", "", nil, errors.New("no test_wallet for mainnet")
		}
	}
	if *action == "create" && *pass == "" {
		return "", "", "", nil, errors.New("wallet create needs a password")
	} else if *action == "recreate" {
		if *pass == "" {
			return "", "", "", nil, errors.New("wallet recreate needs a new password - " +
				"can be different to the previous password")
		}
		if *seed == "" {
			return "", "", "", nil, errors.New("wallet recreate needs the old wallet seed")
		}
		words := strings.SplitN(*seed, " ", 12)
		fmt.Printf("%q (len %d)\n", words, len(words))
		if len(words) != 12 {
			return "", "", "", nil, errors.New("a seed must have 12 words each separated by a space")
		}
		var bad bool
		for _, word := range words {
			if len(word) < 3 {
				fmt.Printf("bad word: '%s'\n", word)
				bad = true
			}
		}
		if bad {
			return "", "", "", nil, errors.New("malformed seed -- did you put extra spaces?")
		}
	}
	cfg, err := makeBasicConfig(*coin, *net)
	return *action, *pass, *seed, cfg, err
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
	action, pass, seed, cfg, err := configure()
	fmt.Println(action, pass, seed)
	if err != nil {
		fmt.Println(err, " - exiting")
		flag.Usage()
		os.Exit(1)
	}
	net := cfg.Params.Name
	fmt.Println(net)

	// make basic client
	ec := btc.NewBtcElectrumClient(cfg)

	if action == "create" {
		err := ec.CreateWallet(pass)
		if err != nil {
			fmt.Println(err)
		}
		os.Exit(1)
	}

	// start client, create node & sync headers
	err = ec.Start(context.Background())
	if err != nil {
		ec.Stop()
		fmt.Printf("%v - exiting.\n%s\n", err, checkSimnetHelp(cfg))
		os.Exit(1)
	}

	// recreate the client's wallet

	if net == "regtest" {
		// for non-mainnet testing recreate a wallet with a known set of keys ..
		// var mnemonic = "jungle pair grass super coral bubble tomato sheriff pulp cancel luggage wagon"
		// err := ec.RecreateWallet(pass, mnemonic)
		err := ec.RecreateWallet(context.TODO(), pass, seed)
		if err != nil {
			fmt.Println(err, " - exiting")
		}
	} else if net == "testnet3" {
		// for non-mainnet testing recreate a wallet with a known set of keys ..
		// err := ec.RecreateWallet("abc", "canyon trip truly ritual lonely quiz romance rose alone journey like bronze")
		err := ec.RecreateWallet(context.TODO(), pass, seed)
		if err != nil {
			fmt.Println(err)
		}
	} else if net == "mainnet" {
		err := ec.RecreateWallet(context.TODO(), pass, seed)
		if err != nil {
			fmt.Println(err)
		}
	}

	ec.Stop()
}
