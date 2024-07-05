package electrumx

// func testNeededServerFns(nodeCtx context.Context, sc *serverConn, network, nettype string) bool {
// 	switch network {
// 	case "Bitcoin":
// 		switch nettype {
// 		case "testnet", "testnet3":
// 			txid := "581d837b8bcca854406dc5259d1fb1e0d314fcd450fb2d4654e78c48120e0135"
// 			_, err := sc.getTransaction(nodeCtx, txid)
// 			if err != nil {
// 				return false
// 			}
// 		case "mainnet":
// 			txid := "f53a8b83f85dd1ce2a6ef4593e67169b90aaeb402b3cf806b37afc634ef71fbc"
// 			_, err := sc.getTransaction(nodeCtx, txid)
// 			if err != nil {
// 				return false
// 			}
// 			// ignore regtest
// 		}

// 		//...

// 	default:
// 		// unknown chain
// 		return false
// 	}
// 	return true
// }
