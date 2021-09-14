package eth

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

type BlockPositionTag string

const (
	Latest  = BlockPositionTag("latest")
	Pending = BlockPositionTag("pending")
)

func TransactionsCountCommand(term ui.Screen, ctx *cli.Context, endpoint rpc.Endpoint) error {
	// validate input
	if !ctx.IsSet(flags.HexParam.Name) {
		return errors.New(fmt.Sprintf("Missing from address in hex --%s", flags.HexParam.Name))
	}
	input := ctx.String(flags.HexParam.Name)
	data, err := hexutil.Decode(input)
	if err != nil {
		return err
	}

	// call
	count, err := TransactionsCount(term, endpoint, common.BytesToAddress(data), Latest)
	if err != nil {
		return err
	}
	term.Output(fmt.Sprintf("%s\n", count))
	return nil
}

func TransactionsCount(term ui.Screen, endpoint rpc.Endpoint, from common.Address, tag BlockPositionTag) (*uint64, error) {
	client := httpclient.NewDefault(term)
	resp := rpc.RpcResultStr{}
	err := rpc.Call(term, client, endpoint, "eth_getTransactionCount", []interface{}{from.Hex(), tag}, &resp)
	if err != nil {
		return nil, err
	}
	val, err := uint256.FromHex(resp.Result)
	if err != nil {
		return nil, err
	}
	count := val.ToBig().Uint64()
	return &count, nil
}
