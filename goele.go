package main

// Run goele as an app

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dev-warrior777/go-electrum-client/client"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

var (
	coins = []string{"btc"} // add as implemented
	nets  = []string{"mainnet", "testnet", "regtest"}
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
	return cfg, nil
}

func configure() (*client.ClientConfig, error) {
	coin := flag.String("coin", "btc", "coin name")
	net := flag.String("net", "regtest", "network type; testnet, mainnet, regtest")
	flag.Parse()
	fmt.Println("coin:", *coin)
	fmt.Println("net:", *net)
	return makeBasicConfig(*coin, *net)
}

func main() {
	cfg, err := configure()
	if err != nil {
		fmt.Println(err, " - exiting")
		os.Exit(1)
	}

	fmt.Println(cfg.Chain, "to be continued")
}
