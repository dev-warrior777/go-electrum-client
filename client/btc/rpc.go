package btc

import (
	"context"
	"strconv"

	"github.com/dev-warrior777/go-electrum-client/client"
	"github.com/dev-warrior777/go-electrum-client/wallet"
	"github.com/go-zoox/jsonrpc"
	"github.com/go-zoox/jsonrpc/server"
	"github.com/go-zoox/logger"
	"github.com/spf13/cast"
)

// For testing only

func rpcServe(ec client.ElectrumClient) {
	s := server.New()

	s.Register("gettip", func(ctx context.Context, params jsonrpc.Params) (jsonrpc.Result, error) {
		logger.Info("params: %s", params)
		logger.Info("Name %s", ec.GetConfig().Params.Name)

		height, synced := ec.Tip()
		tip := strconv.Itoa(int(height))

		return jsonrpc.Result{
			"tip":    tip,
			"synced": synced,
		}, nil
	})

	s.Register("spend", func(ctx context.Context, params jsonrpc.Params) (jsonrpc.Result, error) {
		logger.Info("params: %s", params)
		logger.Info("Name %s", ec.GetConfig().Params.Name)

		amt := cast.ToInt64(params.Get("amount"))
		addr := cast.ToString(params.Get("address"))
		feeType := cast.ToString(params.Get("feeType"))
		var feeLvl wallet.FeeLevel
		switch feeType {
		case "PRIORITY":
			feeLvl = wallet.PRIORITY
		case "NORMAL":
			feeLvl = wallet.NORMAL
		case "ECONOMIC":
			feeLvl = wallet.ECONOMIC
		default:
			feeLvl = wallet.NORMAL
		}

		tx, txid, err := ec.Spend(amt, addr, feeLvl, false)

		if err != nil {
			return jsonrpc.Result{
				"tx":   "",
				"txid": "",
			}, err
		}

		return jsonrpc.Result{
			"tx":   tx,
			"txid": txid,
		}, nil
	})

	s.Run()
}
