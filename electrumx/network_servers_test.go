// This code is available on the terms of the project LICENSE.md file,
// also available online at https://blueoakcouncil.org/license/1.0.0.

package electrumx

import (
	"fmt"
	netIp "net"
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
			DataDir: testDir,
			// This test should probably be split up between strategies for removal
			// NoDeleteKnownPeers, Default
			//
			// However there also needs to be a non-nil logger for the non-default
			// case since adding logging so this is a minor TODO(goele)
			//
			Coin:  "btc",
			Flags: Default, // Strategy
		},
	}
	return net
}

func TestNetworkServersDefaultStrategy(t *testing.T) {
	tmpDir := t.TempDir() // will be cleaned up by T
	net := mkNetwork(tmpDir)
	t.Logf("datadir: %s\n", net.config.DataDir)

	// test nil input
	err := net.addIncomingServers(nil)
	if err == nil {
		// should error "no incoming"
		t.Fatal(err)
	}

	err = net.addIncomingServers(peerNoResults)
	if err == nil {
		t.Fatal(err)
	}
	t.Logf("%v .. OK!", err)

	// add first results to known servers and an empty file
	err = net.addIncomingServers(peerResults)
	if err != nil {
		t.Fatal(err)
	}
	if len(net.knownServers) != 10 {
		t.Fatal("net.knownServers should be 10")
	}

	// update but all the same servers
	err = net.addIncomingServers(peerResults2)
	if err != nil {
		t.Fatal(err)
	}
	if len(net.knownServers) != 10 {
		t.Fatalf("got %d net.knownServers should be 8", len(net.knownServers))
	}

	// update but some of the servers are different and 2 IPs are rejected
	err = net.addIncomingServers(peerResults3)
	if err != nil {
		t.Fatal(err)
	}
	if len(net.knownServers) != 10 {
		t.Fatalf("got %d net.knownServers - 0 is OK", len(net.knownServers))
	}
	storedServers, n, err := net.readServerAddrFile()
	if err != nil {
		t.Fatal(err)
	}
	if n != len(storedServers) {
		t.Logf("got %d stored servers", len(storedServers))
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

	err = net.addIncomingServers(peerResultsIPv6)
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
		t.Fatalf("number known servers is %d, expected %d", len(net.knownServers), n)
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
