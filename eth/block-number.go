package eth

import (
	"fmt"

	"github.com/holiman/uint256"
	"github.com/jaanek/jeth/httpclient"
	"github.com/jaanek/jeth/rpc"
	"github.com/jaanek/jeth/ui"
	"github.com/urfave/cli"
)

func BlockNumberCommand(term ui.Screen, ctx *cli.Context, endpoint rpc.RpcEndpoint) error {
	blockNumber, err := BlockNumber(term, endpoint)
	if err != nil {
		return err
	}
	term.Output(fmt.Sprintf("%s\n", blockNumber))
	return nil
}

func BlockNumber(term ui.Screen, endpoint rpc.RpcEndpoint) (*uint256.Int, error) {
	client := httpclient.NewDefault(term)
	resp := rpc.RpcResultStr{}
	err := rpc.Call(term, client, endpoint, "eth_blockNumber", []interface{}{}, &resp)
	if err != nil {
		return nil, err
	}
	return uint256.FromHex(resp.Result)
}
