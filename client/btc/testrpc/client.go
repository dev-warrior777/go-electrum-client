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

	r, err := c.Call(context.Background(), "gettip", jsonrpc.Params{
		"address": "a-bitcoin-address",
		"amount":  "10000",
		"feeType": "NORMAL",
	})
	if err != nil {
		logger.Errorf("failed to call: %s", err)
		return
	}

	logger.Info("tip: %d", cast.ToInt64(r.Get("tip")))
	logger.Info("synced: %v", cast.ToBool(r.Get("synced")))
}
