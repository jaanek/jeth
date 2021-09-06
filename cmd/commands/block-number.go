package commands

import (
	"errors"
	"fmt"

	"github.com/holiman/uint256"
	"github.com/jaanek/jeth/flags"
	"github.com/jaanek/jeth/httpclient"
	"github.com/jaanek/jeth/rpc"
	"github.com/jaanek/jeth/ui"
	"github.com/urfave/cli"
)

func BlockNumberCommand(term ui.Screen, ctx *cli.Context) error {
	if !ctx.IsSet(flags.RpcUrlFlag.Name) {
		return errors.New(fmt.Sprintf("Missing --%s", flags.RpcUrlFlag.Name))
	}
	endpoint := rpc.NewEndpoint(ctx.String(flags.RpcUrlFlag.Name))
	client := httpclient.NewDefault(term)
	resp, err := rpc.Call(term, client, endpoint, "eth_blockNumber", []string{})
	blockNumber, err := uint256.FromHex(resp.Result)
	if err != nil {
		return err
	}
	term.Output(fmt.Sprintf("%s\n", blockNumber))
	return nil
}
