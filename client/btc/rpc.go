package btc

import (
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/dev-warrior777/go-electrum-client/wallet"
	"github.com/go-zoox/jsonrpc"
	"github.com/go-zoox/jsonrpc/server"
	"github.com/go-zoox/logger"
	"github.com/spf13/cast"
)

// For testing only

func (ec *BtcElectrumClient) RPCServe() {
	s := server.New()

	s.Register("gettip", func(ctx context.Context, params jsonrpc.Params) (jsonrpc.Result, error) {

		height, synced := ec.Tip()
		tip := strconv.Itoa(int(height))

		return jsonrpc.Result{
			"tip":    tip,
			"synced": synced,
		}, nil
	})

	s.Register("listunspent", func(ctx context.Context, params jsonrpc.Params) (jsonrpc.Result, error) {

		utxos, err := ec.ListUnspent()

		if err != nil {
			return jsonrpc.Result{
				"unspents": "",
			}, err
		}

		var sb strings.Builder
		var last = len(utxos) - 1
		for i, utxo := range utxos {
			sb.WriteString(utxo.Op.String())
			sb.WriteString(":")
			sb.WriteString(strconv.Itoa(int(utxo.Value)))
			sb.WriteString(":")
			sb.WriteString(strconv.Itoa(int(utxo.AtHeight)))
			sb.WriteString(":")
			sb.WriteString(hex.EncodeToString(utxo.ScriptPubkey))
			sb.WriteString(":")
			sb.WriteString(strconv.FormatBool(utxo.WatchOnly))
			sb.WriteString(":")
			sb.WriteString(strconv.FormatBool(utxo.Frozen))
			if i != last {
				sb.WriteString("\n")
			}
		}

		return jsonrpc.Result{
			"unspents": sb.String(),
		}, nil
	})

	s.Register("spend", func(ctx context.Context, params jsonrpc.Params) (jsonrpc.Result, error) {
		logger.Info("params: %v", params)

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

		// tx, txid, err := ec.Spend(amt, addr, feeLvl, true)
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

	s.Register("broadcast", func(ctx context.Context, params jsonrpc.Params) (jsonrpc.Result, error) {
		logger.Info("params: %v", params)

		rawTx := cast.ToString(params.Get("rawTx"))

		txid, err := ec.Broadcast(rawTx)

		if err != nil {
			fmt.Printf("Broadcast error: %v\n", err.Error())
			return jsonrpc.Result{
				"txid": "",
			}, err
		}

		return jsonrpc.Result{
			"txid": txid,
		}, nil
	})

	//////////////////
	// run http server
	s.Run()
}
