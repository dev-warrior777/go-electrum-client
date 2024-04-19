package main

// Delete a default test wallet - will not work for mainnet config

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/dev-warrior777/go-electrum-client/client"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

const blockchain_headers = "blockchain_headers"

var (
	coins = []string{"btc"} // add as implemented
	nets  = []string{"testnet", "regtest", "simnet"}
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
		cfg.Testing = true
	case "testnet":
		cfg.Testing = true
	}
	return cfg, nil
}

func configure() (*client.ClientConfig, error) {
	coin := flag.String("coin", "btc", "coin name")
	net := flag.String("net", "regtest", "network type; testnet, regtest")
	flag.Parse()
	fmt.Println("coin:", *coin)
	fmt.Println("net:", *net)
	cfg, err := makeBasicConfig(*coin, *net)
	return cfg, err
}

func main() {
	fmt.Println("Goele", client.GoeleVersion)
	cfg, err := configure()
	if err != nil {
		fmt.Println(err, " - exiting")
		flag.Usage()
		os.Exit(1)
	}
	type walletDb struct {
		db   string
		name string
	}
	walletFiles := []walletDb{
		{
			db:   "bbolt",
			name: "wallet.bdb",
		},
		{
			db:   "sqlite",
			name: "wallet.db",
		},
	}
	for _, w := range walletFiles {
		wallet := path.Join(cfg.DataDir, w.name)
		fmt.Println(wallet)
		if _, err := os.Stat(wallet); errors.Is(err, os.ErrNotExist) {
			fmt.Println(err)
			continue
		}
		if askForConfirmation("remove?") {
			os.Remove(wallet)
		}
	}

	if cfg.Params == &chaincfg.RegressionNetParams {
		headers := path.Join(cfg.DataDir, blockchain_headers)
		fmt.Println(headers)
		if _, err := os.Stat(headers); errors.Is(err, os.ErrNotExist) {
			fmt.Println(err)
			os.Exit(1)
		}
		if askForConfirmation("remove?") {
			os.Remove(headers)
		}
	}
}

func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}
