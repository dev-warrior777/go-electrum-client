package wltbtc

import (
	"github.com/dev-warrior777/go-electrum-client/wallet"
)

// Sign an unsigned transaction with the wallet
func (w *BtcElectrumWallet) SignTx(pw string, info *wallet.SigningInfo) ([]byte, error) {
	// if ok := w.storageManager.IsValidPw(pw); !ok {
	// 	return nil, errors.New("invalid password")
	// }
	// tx := info.UnsignedTx
	// prevOutFetcher := new(txscript.CannedPrevOutputFetcher)

	// for idx, input := range tx.TxIn {
	// 	prevOut := info.PrevOuts[idx]
	// 	output := prevOut.PrevTx.Outputs[idx]
	// 	value := output.Value
	// 	address := output.Address
	// 	pkScript, err := txscript.PayToAddrScript(address)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	key, err := w.txstore.keyManager.GetKeyForScript(address.ScriptAddress())
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	defer key.Zero()
	// 	privKey, err := key.ECPrivKey()
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	sigHashes := txscript.NewTxSigHashes(tx, prevOutFetcher)
	// 	prevOutScriptTy := txscript.GetScriptClass(pkScript)
	// 	switch prevOutScriptTy {
	// 	case txscript.WitnessV0PubKeyHashTy:
	// 		sig, err := txscript.WitnessSignature(tx, sigHashes, idx, value,
	// 			pkScript, txscript.SigHashAll, privKey, true)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		// add witness
	// 		input.SignatureScript = nil
	// 		input.Witness = append(input.Witness, sig...)
	// 	case txscript.PubKeyHashTy:
	// 		sig, err := txscript.SignatureScript(tx, idx,
	// 			pkScript, txscript.SigHashAll, privKey, true)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		// add script sig
	// 		input.SignatureScript = append(input.SignatureScript, sig...)
	// 		input.Witness = nil
	// 	default:
	// 		return nil, fmt.Errorf("signing prev out script type %v unsupported",
	// 			prevOutScriptTy)
	// 	}

	// 	if info.VerifyTx {
	// 		e, err := txscript.NewDebugEngine(
	// 			// pubkey script
	// 			pkScript,
	// 			// refund transaction
	// 			tx,
	// 			// transaction input index
	// 			idx,
	// 			txscript.StandardVerifyFlags,
	// 			txscript.NewSigCache(10),
	// 			txscript.NewTxSigHashes(tx, prevOutFetcher),
	// 			value,
	// 			prevOutFetcher,
	// 			nil)
	// 		if err != nil {
	// 			panic(err)
	// 		}
	// 		// err = e.Execute()
	// 		// if err != nil {
	// 		// 	panic(err)
	// 		// }
	// 		stepDebugScript(e)
	// 	}
	// }

	// b := make([]byte, 0)
	// txOut := bytes.NewBuffer(b)
	// err := tx.Serialize(txOut)
	// if err != nil {
	// 	return nil, err
	// }
	// return txOut.Bytes(), nil

	return nil, nil
}

// func stepDebugScript(e *txscript.Engine) {
// 	fmt.Println("Script 0")
// 	fmt.Println(e.DisasmScript(0))
// 	fmt.Println("Script 1")
// 	fmt.Println(e.DisasmScript(1))
// 	fmt.Printf("End Scripts\n============\n\n")

// 	for {
// 		fmt.Println("---------------------------- STACK --------------------------")
// 		stk := e.GetStack()
// 		for i, item := range stk {
// 			if len(item) > 0 {
// 				fmt.Printf("%d %v\n", i, hex.EncodeToString(item))
// 			} else {
// 				fmt.Printf("%d %s\n", i, "<null>")
// 			}
// 		}
// 		fmt.Println("-------------------------- ALT STACK ------------------------")
// 		astk := e.GetAltStack()
// 		for i, item := range astk {
// 			if len(item) > 0 {
// 				fmt.Printf("%d %v\n", i, hex.EncodeToString(item))
// 			} else {
// 				fmt.Printf("%d %s\n", i, "<null>")
// 			}
// 		}
// 		fmt.Println("--------------------------- Next Op -------------------------")
// 		fmt.Println(e.DisasmPC())
// 		fmt.Println("===========")
// 		script, err := e.DisasmScript(2)
// 		if err == nil {
// 			fmt.Printf("script 2: \n%s\n", script)
// 		}
// 		fmt.Println("..........")

// 		// STEP
// 		done, err := e.Step()
// 		if err != nil {
// 			fmt.Printf("Engine Error: %v\n", err)
// 			os.Exit(2)
// 		}

// 		if done {
// 			fmt.Println("----------------------- Last STACK ------------------------")
// 			stkerr := false
// 			stkerrtxt := ""
// 			stk = e.GetStack()
// 			for i, item := range stk {
// 				fmt.Println(i, hex.EncodeToString(item))
// 				if i == 0 && !bytes.Equal(item, []byte{0x01}) {
// 					stkerr = true
// 					stkerrtxt += "ToS Not '1'"
// 				}
// 				if i > 0 {
// 					stkerr = true
// 					stkerrtxt += " too many stack items left on stack"
// 				}
// 			}
// 			if stkerr {
// 				fmt.Println(stkerrtxt)
// 				os.Exit(3)
// 			}
// 			fmt.Println("--------------------- End Last STACK ------------------------")

// 			// senang
// 			break
// 		}
// 	}
// }
