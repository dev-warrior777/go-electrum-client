package wltbtc

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcwallet/wallet/txrules"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

// Build a transaction that sweeps all coins from a private key into iur wallet.

var stepDebug = true

func (w *BtcElectrumWallet) SweepCoins(coins []wallet.TransactionInput, feeLevel wallet.FeeLevel) (int, *wire.MsgTx, error) {
	var totalValue int64
	var privKeyToSignOutputs map[int]*secp256k1.PrivateKey
	var prevOutScripts map[int][]byte
	var prevOutValues map[int]int64

	scriptForInput := func(address btcutil.Address) []byte {
		script, err := txscript.PayToAddrScript(address)
		if err != nil {
			fmt.Printf("cannot generate a payment script for %s\n", address.String())
		}
		return script
	}

	// walletAddress, err := w.GetUnusedLegacyAddress()
	// if err != nil {
	// 	return -1, nil, err
	// }
	// fmt.Printf("our wallet address: %s\n", walletAddress.String())
	// // make an output script that can spend the inputs to our wallet
	// p2pkhScript, err := txscript.PayToAddrScript(walletAddress)
	// if err != nil {
	// 	return -1, nil, err
	// }

	walletAddressSegwit, err := w.GetUnusedAddress(wallet.RECEIVING)
	if err != nil {
		return -1, nil, err
	}
	fmt.Printf("our wallet address: %s\n", walletAddressSegwit.String())
	// make an output script that can spend the inputs to our wallet
	p2wpkhScript, err := txscript.PayToAddrScript(walletAddressSegwit)
	if err != nil {
		return -1, nil, err
	}

	sweepTx := wire.NewMsgTx(2 /*wire.TxVersion*/)

	privKeyToSignOutputs = make(map[int]*secp256k1.PrivateKey)
	prevOutScripts = make(map[int][]byte)
	prevOutValues = make(map[int]int64)

	for i, coin := range coins {
		fmt.Println(coin.String())
		totalValue += coin.Value
		prevOutValues[i] = coin.Value
		prevOutScripts[i] = scriptForInput(coin.LinkedAddress)
		privKeyToSignOutputs[i] = coin.KeyPair.PrivKey
		// make a basic input from coin
		input := wire.NewTxIn(coin.Outpoint, []byte{}, [][]byte{})
		sweepTx.TxIn = append(sweepTx.TxIn, input)
	}

	// get size estimate without the output //TODO: non-segwit ins & out
	sweepSize := txSerialSizeEst(sweepTx)

	// get the fee
	feePerKB := btcutil.Amount(w.GetFeePerByte(feeLevel)) * 1000
	fee := txrules.FeeForSerializeSize(feePerKB, sweepSize)

	// add single output
	outValue := totalValue - int64(fee)
	// sweepTx.AddTxOut(wire.NewTxOut(outValue, p2pkhScript))
	sweepTx.AddTxOut(wire.NewTxOut(outValue, p2wpkhScript))
	err = txrules.CheckOutput(sweepTx.TxOut[0], feePerKB)
	if err != nil {
		return -1, &wire.MsgTx{}, nil
	}

	// sign

	// NewTxSigHashes uses the PrevOutFetcher only for detecting a taproot
	// output, so we can provide a dummy.
	prevOutFetcher := new(txscript.CannedPrevOutputFetcher)
	sigHashes := txscript.NewTxSigHashes(sweepTx, prevOutFetcher)

	for idx, input := range sweepTx.TxIn {
		if txscript.IsPayToWitnessPubKeyHash(prevOutScripts[idx]) {
			for idx, input := range sweepTx.TxIn {
				sig, err := txscript.WitnessSignature(sweepTx, sigHashes, idx, prevOutValues[idx],
					prevOutScripts[idx], txscript.SigHashAll, privKeyToSignOutputs[idx], true)
				if err != nil {
					return -1, nil, err
				}
				// add witness
				input.Witness = append(input.Witness, sig...)
			}
		} else if txscript.IsPayToPubKeyHash(prevOutScripts[idx]) {
			sig, err := txscript.SignatureScript(sweepTx, idx,
				prevOutScripts[idx], txscript.SigHashAll, privKeyToSignOutputs[idx], true)
			if err != nil {
				return -1, nil, err
			}
			// add script sig
			input.SignatureScript = append(input.SignatureScript, sig...)
			input.Witness = nil
		}
		//TODO: switch
	}
	// Use the Debug Stepper OR the Execute option. NOT both with same egine instance
	e, err := txscript.NewDebugEngine(
		// previous output pubkey script (from input)
		prevOutScripts[0],
		// sweep transaction
		sweepTx,
		// transaction input index
		0,
		txscript.StandardVerifyFlags,
		txscript.NewSigCache(10),
		txscript.NewTxSigHashes(sweepTx, prevOutFetcher),
		totalValue,
		prevOutFetcher,
		nil)
	if err != nil {
		panic(err)
	}
	if stepDebug {
		stepDebugScript(e)
	} else {
		err = e.Execute()
		if err != nil {
			fmt.Printf("Engine Error: %v\n", err)
			os.Exit(1)
		}
	}

	return -1, sweepTx, nil
}

func stepDebugScript(e *txscript.Engine) {
	fmt.Println("Script 0")
	fmt.Println(e.DisasmScript(0))
	fmt.Println("Script 1")
	fmt.Println(e.DisasmScript(1))
	fmt.Printf("End Scripts\n============\n\n")

	for {
		fmt.Println("---------------------------- STACK --------------------------")
		stk := e.GetStack()
		for i, item := range stk {
			if len(item) > 0 {
				fmt.Printf("%d %v\n", i, hex.EncodeToString(item))
			} else {
				fmt.Printf("%d %s\n", i, "<null>")
			}
		}
		fmt.Println("-------------------------- ALT STACK ------------------------")
		astk := e.GetAltStack()
		for i, item := range astk {
			if len(item) > 0 {
				fmt.Printf("%d %v\n", i, hex.EncodeToString(item))
			} else {
				fmt.Printf("%d %s\n", i, "<null>")
			}
		}
		fmt.Println("--------------------------- Next Op -------------------------")
		fmt.Println(e.DisasmPC())
		fmt.Println("===========")
		script, err := e.DisasmScript(2)
		if err == nil {
			fmt.Printf("script 2: \n%s\n", script)
		}
		fmt.Println("..........")

		// STEP
		done, err := e.Step()
		if err != nil {
			fmt.Printf("Engine Error: %v\n", err)
			os.Exit(2)
		}

		if done {
			fmt.Println("----------------------- Last STACK ------------------------")
			stkerr := false
			stkerrtxt := ""
			stk = e.GetStack()
			for i, item := range stk {
				fmt.Println(i, hex.EncodeToString(item))
				if i == 0 && !bytes.Equal(item, []byte{0x01}) {
					stkerr = true
					stkerrtxt += "ToS Not '1'"
				}
				if i > 0 {
					stkerr = true
					stkerrtxt += " too many stack items left on stack"
				}
			}
			if stkerr {
				fmt.Println(stkerrtxt)
				os.Exit(3)
			}
			fmt.Println("--------------------- End Last STACK ------------------------")

			// senang
			break
		}
	}
}

//----------------------------------------------------------------------
//----------------------------------------------------------------------
//----------------------------------------------------------------------

// If it is a p2sh multisig, the redeemScript must be included
// Tx is returned ready to broadcast. Only the client can broadcast (via Node)
// func (w *BtcElectrumWallet) SweepAddress(ins []wallet.TransactionInput, address btcutil.Address, key *hdkeychain.ExtendedKey, redeemScript []byte, feeLevel wallet.FeeLevel) (*wire.MsgTx, error) {

// 	return nil, wallet.ErrWalletFnNotImplemented // FIXME: SweepAddress prevOutputFInder

// if address == nil {
// 	return nil, errors.New("empty address")
// }

// script, err := txscript.PayToAddrScript(address)
// if err != nil {
// 	return nil, err
// }

// var val int64
// var inputs []*wire.TxIn
// additionalPrevScripts := make(map[wire.OutPoint][]byte)
// for _, in := range ins {
// 	val += in.Value
// 	ch, err := chainhash.NewHashFromStr(hex.EncodeToString(in.OutpointHash))
// 	if err != nil {
// 		return nil, err
// 	}
// 	script, err := txscript.PayToAddrScript(in.LinkedAddress)
// 	if err != nil {
// 		return nil, err
// 	}
// 	outpoint := wire.NewOutPoint(ch, in.OutpointIndex)
// 	input := wire.NewTxIn(outpoint, []byte{}, [][]byte{})
// 	inputs = append(inputs, input)
// 	additionalPrevScripts[*outpoint] = script
// }
// out := wire.NewTxOut(val, script)

// txType := P2PKH
// if redeemScript != nil {
// 	txType = P2SH_1of2_Multisig
// 	_, err := LockTimeFromRedeemScript(redeemScript)
// 	if err == nil {
// 		txType = P2SH_Multisig_Timelock_1Sig
// 	}
// }
// estimatedSize := EstimateSerializeSize(len(ins), []*wire.TxOut{out}, false, txType)

// // Calculate the fee
// feePerByte := int(w.GetFeePerByte(feeLevel))
// fee := estimatedSize * feePerByte

// outVal := val - int64(fee)
// if outVal < 0 {
// 	outVal = 0
// }
// out.Value = outVal

// tx := &wire.MsgTx{
// 	Version:  wire.TxVersion,
// 	TxIn:     inputs,
// 	TxOut:    []*wire.TxOut{out},
// 	LockTime: 0,
// }

// // BIP 69 sorting
// txsort.InPlaceSort(tx)

// // Sign tx
// privKey, err := key.ECPrivKey()
// if err != nil {
// 	return nil, err
// }
// pk := privKey.PubKey().SerializeCompressed()
// addressPub, err := btcutil.NewAddressPubKey(pk, w.params)

// getKey := txscript.KeyClosure(func(addr btcutil.Address) (*btcec.PrivateKey, bool, error) {
// 	if addressPub.EncodeAddress() == addr.EncodeAddress() {
// 		wif, err := btcutil.NewWIF(privKey, w.params, true)
// 		if err != nil {
// 			return nil, false, err
// 		}
// 		return wif.PrivKey, wif.CompressPubKey, nil
// 	}
// 	return nil, false, errors.New("not found")
// })
// getScript := txscript.ScriptClosure(func(addr btcutil.Address) ([]byte, error) {
// 	if redeemScript == nil {
// 		return []byte{}, nil
// 	}
// 	return redeemScript, nil
// })

// // Check if time locked
// var timeLocked bool
// if redeemScript != nil {
// 	rs := redeemScript
// 	if rs[0] == txscript.OP_IF {
// 		timeLocked = true
// 		tx.Version = 2
// 		for _, txIn := range tx.TxIn {
// 			locktime, err := LockTimeFromRedeemScript(redeemScript)
// 			if err != nil {
// 				return nil, err
// 			}
// 			txIn.Sequence = locktime
// 		}
// 	}
// }

// //FIXME:
// fetcher := txscript.NewCannedPrevOutputFetcher(
// 	utxOut.PkScript, utxOut.Value,
// )
// hashes := txscript.NewTxSigHashes(tx, fetcher)
// for i, txIn := range tx.TxIn {
// 	if redeemScript == nil {
// 		prevOutScript := additionalPrevScripts[txIn.PreviousOutPoint]
// 		script, err := txscript.SignTxOutput(w.params,
// 			tx, i, prevOutScript, txscript.SigHashAll, getKey,
// 			getScript, txIn.SignatureScript)
// 		if err != nil {
// 			return nil, errors.New("failed to sign transaction")
// 		}
// 		txIn.SignatureScript = script
// 	} else {
// 		sig, err := txscript.RawTxInWitnessSignature(tx, hashes, i, ins[i].Value, redeemScript, txscript.SigHashAll, privKey)
// 		if err != nil {
// 			return nil, err
// 		}
// 		var witness wire.TxWitness
// 		if timeLocked {
// 			witness = wire.TxWitness{sig, []byte{}}
// 		} else {
// 			witness = wire.TxWitness{[]byte{}, sig}
// 		}
// 		witness = append(witness, redeemScript)
// 		txIn.Witness = witness
// 	}
// }

// ready to broadcast -- only client can broadcast

// return &tx, nil
// }
