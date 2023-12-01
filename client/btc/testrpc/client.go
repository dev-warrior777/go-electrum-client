package main

import (
	"context"

	"github.com/go-zoox/core-utils/cast"
	"github.com/go-zoox/jsonrpc"
	zoojc "github.com/go-zoox/jsonrpc/client"
	"github.com/go-zoox/logger"
)

func main() {
	c := zoojc.New("http://localhost:8080")

	// r, err := c.Call(context.Background(), "gettip", jsonrpc.Params{})
	// if err != nil {
	// 	logger.Errorf("failed to call: %s", err)
	// 	return
	// }

	// logger.Info("tip: %d", cast.ToInt64(r.Get("tip")))
	// logger.Info("txid: %v", cast.ToBool(r.Get("synced")))

	r, err := c.Call(context.Background(), "spend", jsonrpc.Params{
		"address": "bcrt1q322tg0y2hzyp9zztr7d2twdclhqg88anvzxwwr",
		"amount":  "100000000",
		"feeType": "NORMAL",
	})
	if err != nil {
		logger.Errorf("failed to call: %s", err)
		return
	}

	logger.Info("tx: %d", cast.ToString(r.Get("tx")))
	logger.Info("txid: %v", cast.ToString(r.Get("txid")))
}
