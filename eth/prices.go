package eth

import (
	"fmt"

	"github.com/holiman/uint256"
	"github.com/jaanek/jeth/flags"
	"github.com/jaanek/jeth/httpclient"
	"github.com/jaanek/jeth/rpc"
	"github.com/jaanek/jeth/ui"
	"github.com/ledgerwatch/erigon/params"
	"github.com/urfave/cli"
)

func GasPriceCommand(term ui.Screen, ctx *cli.Context, endpoint rpc.RpcEndpoint) error {
	gasPrice, err := GasPrice(term, endpoint)
	if err != nil {
		return err
	}
	if ctx.IsSet(flags.Gwei.Name) {
		gasPrice = new(uint256.Int).Div(gasPrice, new(uint256.Int).SetUint64(params.GWei))
	}
	term.Output(fmt.Sprintf("%s\n", gasPrice))
	return nil
}

func GasPrice(term ui.Screen, endpoint rpc.RpcEndpoint) (*uint256.Int, error) {
	client := httpclient.NewDefault(term)
	resp := rpc.RpcResultStr{}
	err := rpc.Call(term, client, endpoint, "eth_gasPrice", []interface{}{}, &resp)
	if err != nil {
		return nil, err
	}
	return uint256.FromHex(resp.Result)
}

func MaxPriorityFeePerGasCommand(term ui.Screen, ctx *cli.Context, endpoint rpc.RpcEndpoint) error {
	maxTip, err := MaxPriorityFeePerGas(term, endpoint)
	if err != nil {
		return err
	}
	if ctx.IsSet(flags.Gwei.Name) {
		maxTip = new(uint256.Int).Div(maxTip, new(uint256.Int).SetUint64(params.GWei))
	}
	term.Output(fmt.Sprintf("%s\n", maxTip))
	return nil
}

func MaxPriorityFeePerGas(term ui.Screen, endpoint rpc.RpcEndpoint) (*uint256.Int, error) {
	client := httpclient.NewDefault(term)
	resp := rpc.RpcResultStr{}
	err := rpc.Call(term, client, endpoint, "eth_maxPriorityFeePerGas", []interface{}{}, &resp)
	if err != nil {
		return nil, err
	}
	return uint256.FromHex(resp.Result)
}
