package main

import (
	"fmt"
	"log"
	"net/rpc"
	"strings"

	"github.com/spf13/cast"
)

// func main() {
// 	client, err := rpc.DialHTTP("tcp", "127.0.0.1:8888")
// 	if err != nil {
// 		log.Fatal("dialing:", err)
// 	}
// 	var request = make(map[string]string)
// 	request["a"] = "A"
// 	request["b"] = "B"
// 	request["c"] = "C"
// 	var response = make(map[string]string)
// 	err = client.Call("Ec.RPCEcho", &request, &response)
// 	if err != nil {
// 		log.Fatal("Ec.RPCEcho:", err)
// 	}
// 	fmt.Printf("client response %v\n", response)
// }

// func main() {
// 	client, err := rpc.DialHTTP("tcp", "127.0.0.1:8888")
// 	if err != nil {
// 		log.Fatal("dialing:", err)
// 	}
// 	var request = make(map[string]string)
// 	var response = make(map[string]string)
// 	err = client.Call("Ec.RPCTip", &request, &response)
// 	if err != nil {
// 		log.Fatal("Ec.RPCTip:", err)
// 	}
// 	fmt.Printf("client response %v\n", response)
// }

func main() {
	client, err := rpc.DialHTTP("tcp", "127.0.0.1:8888")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	var request = make(map[string]string)
	var response = make(map[string]string)
	err = client.Call("Ec.RPCListUnspent", &request, &response)
	if err != nil {
		log.Fatal("Ec.RPCListUnspent:", err)
	}
	fmt.Printf("client response %v\n", response)
	allUnspents := cast.ToString(response["unspents"])

	unspents := strings.Split(allUnspents, "\n")

	var us []string
	fmt.Println("[")
	for _, unspent := range unspents {
		us = strings.Split(unspent, ":")
		fmt.Println(" {")
		fmt.Println("   txid:", us[0])
		fmt.Println("   vout:", us[1])
		fmt.Println("   value:", us[2])
		fmt.Println("   spendheight:", us[3])
		fmt.Println("   script:", us[4])
		fmt.Println("   watch only:", us[5])
		fmt.Println("   frozen:", us[6])
		fmt.Println(" }")
	}
	fmt.Println("]")
}

////////////////////////////////////
// Old
//////
// // =====
// // Spend
// // =====
// r, err := c.Call(context.Background(), "spend", jsonrpc.Params{
// 	"address": "bcrt1q322tg0y2hzyp9zztr7d2twdclhqg88anvzxwwr",
// 	"amount":  "100000000",
// 	"feeType": "NORMAL",
// })
// if err != nil {
// 	logger.Errorf("failed to call: %s", err)
// 	return
// }

// logger.Info("tx: %d", cast.ToString(r.Get("tx")))
// logger.Info("txid: %v", cast.ToString(r.Get("txid")))

// // =========
// // Broadcast
// // =========
// r, err := c.Call(context.Background(), "broadcast", jsonrpc.Params{
// 	"rawTx": "0100000001ea2d00243734672280308a112cc5b77ec6b7550c522d4a5a1578fd2edd92f65b000000006b483045022100cd5ef583ade6acd1fdd9650b52d4b8a3a50d1d061cba6fdf3840083f43ac840d022010d8731fe53930e17ba7775378528e689af2d21e65b4fab7fd6dcd949553b772012102cb969af83427bfb1d271a7eb16f7fa3d16794a93369d0da293f721e925af9135000000000200e1f505000000001600148a94b43c8ab88812884b1f9aa5b9b8fdc0839fb3a8f66528000000001976a914dd3c22b42d29ea8ab7ec454e8bce628a07200ccd88ac00000000",
// })
// if err != nil {
// 	logger.Errorf("failed to call: %s", err)
// 	return
// }

// logger.Info("txid: %v", cast.ToString(r.Get("txid")))