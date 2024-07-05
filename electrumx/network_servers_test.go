package electrumx

import (
	"os"
	"testing"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

var peerNoResults = []*peersResult{}

var peerResults = []*peersResult{
	{
		Addr:  "gsw6sn27quwf6u3swgra6o7lrp5qau6kt3ymuyoxgkth6wntzm2bjwyd:onion",
		Host:  "gsw6sn27quwf6u3swgra6o7lrp5qau6kt3ymuyoxgkth6wntzm2bjwyd.onion",
		Feats: []string{"v1.5.3", "s51002", "t51001"},
	},
	{
		Addr:  "203.132.94.196",
		Host:  "testnet.aranguren.org",
		Feats: []string{"v1.5.3", "s51002", "t51001"},
	},
	{
		Addr:  "203.132.94.196",
		Host:  "testnet.aranguren.org",
		Feats: []string{"v1.5", "s51002", "t51001"},
	},
	{
		Addr:  "3tc6nefii2fwoc66dqvrwcyj64dd3r35ihgxvp4u37itsopns5fjtead.onion",
		Host:  "3tc6nefii2fwoc66dqvrwcyj64dd3r35ihgxvp4u37itsopns5fjtead.onion",
		Feats: []string{"v1.5", "s51002", "t51001"},
	},
}

var peerResults2 = []*peersResult{ // identical to peersResult above
	{
		Addr:  "gsw6sn27quwf6u3swgra6o7lrp5qau6kt3ymuyoxgkth6wntzm2bjwyd:onion",
		Host:  "gsw6sn27quwf6u3swgra6o7lrp5qau6kt3ymuyoxgkth6wntzm2bjwyd.onion",
		Feats: []string{"v1.5.3", "s51002", "t51001"},
	},
	{
		Addr:  "203.132.94.196",
		Host:  "testnet.aranguren.org",
		Feats: []string{"v1.5.3", "s51002", "t51001"},
	},
	{
		Addr:  "203.132.94.196",
		Host:  "testnet.aranguren.org",
		Feats: []string{"v1.5", "s51002", "t51001"},
	},
	{
		Addr:  "3tc6nefii2fwoc66dqvrwcyj64dd3r35ihgxvp4u37itsopns5fjtead.onion",
		Host:  "3tc6nefii2fwoc66dqvrwcyj64dd3r35ihgxvp4u37itsopns5fjtead.onion",
		Feats: []string{"v1.5", "s51002", "t51001"},
	},
}

var peerResults3 = []*peersResult{ // mixed: some in the above some new
	{
		Addr:  "new ONION:PORT",
		Host:  "new6sn27quwf6u3swgra6o7lrp5qau6kt3ymuyoxgkth6wntzm2bjwyd.onion",
		Feats: []string{"v1.5.3", "s51002", "t51001"},
	},
	{
		Addr:  "203.132.94.196",
		Host:  "testnet.aranguren.org",
		Feats: []string{"v1.5", "s51002", "t51001"},
	},
	{
		Addr:  "new IP:PORT",
		Host:  "testnet.aranguren.org",
		Feats: []string{"v1.5.3", "s51002", "t51001"},
	},
	{
		Addr:  "3tc6nefii2fwoc66dqvrwcyj64dd3r35ihgxvp4u37itsopns5fjtead.onion",
		Host:  "3tc6nefii2fwoc66dqvrwcyj64dd3r35ihgxvp4u37itsopns5fjtead.onion",
		Feats: []string{"v1.5", "s51002", "t51001"},
	},
}

func mkNetwork(testDir string) *Network {
	net := &Network{
		config: &ElectrumXConfig{
			Chain:   wallet.Bitcoin,
			Params:  &chaincfg.MainNetParams,
			DataDir: testDir,
		},
	}
	return net
}

func TestNetworkServers(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "tns_")
	defer os.RemoveAll(tmpDir)
	net := mkNetwork(tmpDir)
	t.Logf("datadir: %s\n", net.config.DataDir)

	// test nil input
	err := net.addIncomingservers(nil)
	if err == nil {
		// should error "no incoming"
		t.Fatal(err)
	}

	err = net.addIncomingservers(peerNoResults)
	if err == nil {
		t.Fatal(err)
	}
	t.Logf("%v .. OK!", err)

	// add first results to known servers and an empty file
	err = net.addIncomingservers(peerResults)
	if err != nil {
		t.Fatal(err)
	}
	if len(net.knownServers) != 8 {
		t.Fatal("net.knownServers should be 8")
	}

	// update but all the same servers
	err = net.addIncomingservers(peerResults2)
	if err != nil {
		t.Fatal(err)
	}
	if len(net.knownServers) != 8 {
		t.Fatalf("got %d net.knownServers should be 8", len(net.knownServers))
	}

	// update but some of the servers are different
	err = net.addIncomingservers(peerResults3)
	if err != nil {
		t.Fatal(err)
	}
	if len(net.knownServers) != 12 {
		t.Fatalf("got %d net.knownServers should be 12", len(net.knownServers))
	}
	storedServers, n, err := net.readServerAddrFile()
	if err != nil {
		t.Fatal(err)
	}
	if n != 12 {
		t.Fatalf("got %d stored servers should be 12", len(storedServers))
	}

	// remove one by one
	for _, server := range net.knownServers {
		err = net.removeServer(server)
		if err != nil {
			t.Fatal(err)
		}
	}

	err = net.removeServer(nil)
	if err != nil {
		t.Fatal(err)
	}
}
