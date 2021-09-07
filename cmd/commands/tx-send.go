package commands

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

func SendTransactionCommand(term ui.Screen, ctx *cli.Context, endpoint rpc.RpcEndpoint) error {
	// try to read transaction from std input
	var stdInStr string
	if ctx.IsSet(flags.StdIn.Name) {
		stdInStr = StdInReadAll()
		fmt.Printf("All input: %s\n", string(stdInStr))
	}

	// validate input
	var input string
	if ctx.IsSet(flags.HexParam.Name) && stdInStr == "" {
		input = ctx.String(flags.HexParam.Name)
	} else if stdInStr != "" {
		input = stdInStr
	} else {
		return errors.New(fmt.Sprintf("Missing signed tx in hex --%s", flags.HexParam.Name))
	}
	data, err := hexutil.Decode(input)
	if err != nil {
		return err
	}
	tx, err := types.DecodeTransaction(rlp.NewStream(bytes.NewReader(data), uint64(len(data))))
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
	term.Logf("gas: %v\n", tx.GetGas())
	term.Logf("gasPrice: %v\n", tx.GetPrice())

	// send tx
	hash, err := SendTransaction(term, endpoint, input)
	if err != nil {
		return err
	}
	// wait for tx receipt
	c, _ := context.WithTimeout(context.Background(), 120*time.Second)
	receipt, err := WaitTransactionReceipt(c, term, endpoint, hash)
	if err != nil {
		return err
	}
	term.Output(fmt.Sprintf("hash: %s\n", hash))
	term.Output(fmt.Sprintf("receipt: %v\n", receipt))
	return nil
}

// returns tx hash
func SendTransaction(term ui.Screen, endpoint rpc.RpcEndpoint, rawSignedTx string) (string, error) {
	client := httpclient.NewDefault(term)
	resp := rpc.RpcResultStr{}
	err := rpc.Call(term, client, endpoint, "eth_sendRawTransaction", StringsToInterfaces([]string{rawSignedTx}), &resp)
	if err != nil {
		return "", err
	}
	return resp.Result, nil
}
