package eth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jaanek/jeth/flags"
	"github.com/jaanek/jeth/httpclient"
	"github.com/jaanek/jeth/rpc"
	"github.com/jaanek/jeth/ui"
	"github.com/ledgerwatch/erigon/common/hexutil"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/rlp"
	"github.com/urfave/cli"
)

func SendTransactionCommand(term ui.Screen, ctx *cli.Context, endpoint rpc.Endpoint) error {
	// validate input
	var rawTxStr string
	if ctx.IsSet(flags.TxParam.Name) {
		rawTxStr = ctx.String(flags.TxParam.Name)
	} else if flags.FlagRawTx != nil && *flags.FlagRawTx != "" {
		rawTxStr = *flags.FlagRawTx
	} else {
		return errors.New(fmt.Sprintf("Missing signed tx in --%s", flags.TxParam.Name))
	}
	rawTx, err := hexutil.Decode(rawTxStr)
	if err != nil {
		return err
	}
	tx, err := types.DecodeTransaction(rlp.NewStream(bytes.NewReader(rawTx), uint64(len(rawTx))))
	if err != nil {
		return err
	}
	// compare that provided tx chain id is same as endpoint chain id
	endpointChainId, err := ChainId(term, endpoint)
	if err != nil {
		return err
	}
	if tx.GetChainID().Cmp(endpointChainId) != 0 {
		return errors.New(fmt.Sprintf("endpoint chain-id: %v not same as tx chain-id: %v", endpointChainId, tx.GetChainID()))
	}
	term.Print(fmt.Sprintf("Sending tx to: %s (nonce: %d, gas: %d)", endpoint.Url(), tx.GetNonce(), tx.GetGas()))
	term.Logf("gas: %v\n", tx.GetGas())
	term.Logf("gasPrice: %v\n", tx.GetPrice())

	// send tx
	hash, err := SendTransaction(term, endpoint, rawTx)
	if err != nil {
		return err
	}
	term.Print(fmt.Sprintf("Sent tx. Hash: %s Waiting for confirmation...", hash))

	// wait for tx receipt
	c, _ := context.WithTimeout(context.Background(), 120*time.Second)
	receipt, err := WaitTransactionReceipt(c, term, endpoint, hash)
	if err != nil {
		return err
	}
	term.Print(fmt.Sprintf("Received receipt: %+v", receipt))
	return nil
}

// returns tx hash
func SendTransaction(term ui.Screen, endpoint rpc.Endpoint, rawSignedTx []byte) (string, error) {
	tx := hexutil.Encode(rawSignedTx)
	client := httpclient.NewDefault(term)
	resp := rpc.RpcResultStr{}
	err := rpc.Call(term, client, endpoint, "eth_sendRawTransaction", StringsToInterfaces([]string{tx}), &resp)
	if err != nil {
		return "", err
	}
	return resp.Result, nil
}
