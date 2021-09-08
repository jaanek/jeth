package commands

import (
	"errors"
	"fmt"

	"github.com/holiman/uint256"
	"github.com/jaanek/jeth/flags"
	"github.com/jaanek/jeth/httpclient"
	"github.com/jaanek/jeth/rpc"
	"github.com/jaanek/jeth/ui"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/hexutil"
	"github.com/urfave/cli"
)

func GetAccountBalanceCommand(term ui.Screen, ctx *cli.Context, endpoint rpc.RpcEndpoint) error {
	// validate args
	if !ctx.IsSet(flags.HexParam.Name) {
		return errors.New(fmt.Sprintf("Missing address --%s", flags.HexParam.Name))
	}
	input := ctx.String(flags.HexParam.Name)
	data, err := hexutil.Decode(input)
	if err != nil {
		return err
	}
	fromAddr := common.BytesToAddress(data)

	// call
	balance, err := GetAccountBalance(term, endpoint, fromAddr)
	if err != nil {
		return err
	}
	if ctx.IsSet(flags.Verbose.Name) {
		term.Output(fmt.Sprintf("%s: %v\n", input, balance))
		return nil
	}
	term.Output(fmt.Sprintf("%v\n", balance))
	return nil
}

func GetAccountBalance(term ui.Screen, endpoint rpc.RpcEndpoint, fromAddr common.Address) (*uint256.Int, error) {
	client := httpclient.NewDefault(term)
	resp := rpc.RpcResultStr{}
	err := rpc.Call(term, client, endpoint, "eth_getBalance", StringsToInterfaces([]string{fromAddr.Hex(), "latest"}), &resp)
	if err != nil {
		return nil, err
	}
	return uint256.FromHex(resp.Result)
}
