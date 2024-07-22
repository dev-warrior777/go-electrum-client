package electrumx

import (
	"fmt"
	netIp "net"
	"os"
	"testing"
)

var peerNoResults = []*peersResult{}

var peerResults = []*peersResult{
	{
		Addr:  "gsw6sn27quwf6u3swgra6o7lrp5qau6kt3ymuyoxgkth6wntzm2bjwyd.onion",
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
	{
		Addr:  "2600:1900:40b0:3af2:0:5::",
		Host:  "2600:1900:40b0:3af2:0:5::",
		Feats: []string{"v1.4.2", "s50002", "t50001"},
	},
}

var peerResults2 = []*peersResult{ // identical to peersResult above
	{
		Addr:  "gsw6sn27quwf6u3swgra6o7lrp5qau6kt3ymuyoxgkth6wntzm2bjwyd.onion",
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
	{
		Addr:  "2600:1900:40b0:3af2:0:5::",
		Host:  "2600:1900:40b0:3af2:0:5::",
		Feats: []string{"v1.4.2", "s50002", "t50001"},
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

var peerResultsIPv6 = []*peersResult{ // mixed: some in the above some new
	{
		Addr:  "2600:1900:40f0:964b::",
		Host:  "testIPv6.test.org",
		Feats: []string{"v1.5.3", "s50002", "t50001"},
	},
	{
		Addr:  "2600:1901:81c0:6a5:0:3::",
		Host:  "2600:1901:81c0:6a5:0:3::",
		Feats: []string{"v1.5.3", "s50002", "t50001"},
	},
}

func mkNetwork(testDir string) *Network {
	net := &Network{
		config: &ElectrumXConfig{
			// Chain:   wallet.Bitcoin,
			// Params:  &chaincfg.MainNetParams,
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
	if len(net.knownServers) != 10 {
		t.Fatal("net.knownServers should be 8")
	}

	// update but all the same servers
	err = net.addIncomingservers(peerResults2)
	if err != nil {
		t.Fatal(err)
	}
	if len(net.knownServers) != 10 {
		t.Fatalf("got %d net.knownServers should be 8", len(net.knownServers))
	}

	// update but some of the servers are different and 2 IPs are rejected
	err = net.addIncomingservers(peerResults3)
	if err != nil {
		t.Fatal(err)
	}
	if len(net.knownServers) != 10 {
		t.Fatalf("got %d net.knownServers should be 10", len(net.knownServers))
	}
	storedServers, n, err := net.readServerAddrFile()
	if err != nil {
		t.Fatal(err)
	}
	if n != 10 {
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

	err = net.addIncomingservers(peerResultsIPv6)
	if err != nil {
		t.Fatal(err)
	}

	n = 0
	for _, ks := range net.knownServers {
		h, _, _ := netIp.SplitHostPort(ks.Address)
		if nil == netIp.ParseIP(h) {
			fmt.Printf("bad ip addr: %s\n", ks.Address)
			continue
		}
		n++
	}
	if len(net.knownServers) != n {
		t.Fatal()
	}
}

// func dumpPres(pr *peersResult) {
// 	fmt.Printf("Addr  %s\n", pr.Addr)
// 	fmt.Printf("Host  %s\n", pr.Host)
// 	fmt.Printf("Feats %s\n", pr.Feats)
// 	fmt.Println()
// }

// func dumpServer(sa *serverAddr) {
// 	fmt.Printf("Net     %s\n", sa.Net)
// 	fmt.Printf("Address %s\n", sa.Address)
// 	fmt.Printf("Host    %s\n", sa.Host)
// 	fmt.Printf("IsOnion %v\n", sa.IsOnion)
// 	fmt.Printf("Version %s\n", sa.Version)
// 	fmt.Printf("Caps    %s\n", sa.Caps)
// 	fmt.Println()
// }
