package commands

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/holiman/uint256"
	"github.com/jaanek/jeth/flags"
	"github.com/jaanek/jeth/httpclient"
	"github.com/jaanek/jeth/rpc"
	"github.com/jaanek/jeth/ui"
	"github.com/ledgerwatch/erigon/common/hexutil"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/params"
	"github.com/ledgerwatch/erigon/rlp"
	"github.com/urfave/cli"
)

type EstimateGasParam struct {
	From     string  `json:"from"`
	To       string  `json:"to"`
	Value    string  `json:"value"`
	Data     string  `json:"data"`
	Gas      *string `json:"gas"`
	GasPrice *string `json:"gasPrice"`
}

func EstimateGasCommand(term ui.Screen, ctx *cli.Context, endpoint rpc.RpcEndpoint) error {
	// validate input
	if !ctx.IsSet(flags.HexParam.Name) {
		return errors.New(fmt.Sprintf("Missing signed tx in hex --%s", flags.HexParam.Name))
	}
	input := ctx.String(flags.HexParam.Name)
	data, err := hexutil.Decode(input)
	if err != nil {
		return err
	}
	tx, err := types.DecodeTransaction(rlp.NewStream(bytes.NewReader(data), uint64(len(data))))
	if err != nil {
		return err
	}
	gas, err := EstimateGas(term, endpoint, tx)
	if err != nil {
		return err
	}
	term.Output(fmt.Sprintf("%s\n", gas))
	return nil
}

func EstimateGas(term ui.Screen, endpoint rpc.RpcEndpoint, tx types.Transaction) (*uint256.Int, error) {
	// resolve sender from tx
	var signer *types.Signer
	var chainId = tx.GetChainID().ToBig()
	switch chainId.Uint64() {
	case 1:
		signer = types.LatestSigner(params.MainnetChainConfig)
	case 5:
		signer = types.LatestSigner(params.GoerliChainConfig)
	default:
		signer = types.LatestSignerForChainID(chainId)
	}
	sender, err := tx.Sender(*signer)
	if err != nil {
		return nil, err
	}

	// call endpoint
	client := httpclient.NewDefault(term)
	resp := rpc.RpcResultStr{}
	err = rpc.Call(term, client, endpoint, "eth_estimateGas", []interface{}{EstimateGasParam{
		From:  sender.Hex(),
		To:    tx.GetTo().Hex(),
		Value: tx.GetValue().Hex(),
		Data:  hexutil.Encode(tx.GetData()),
	}, "latest"}, &resp)
	if err != nil {
		return nil, err
	}
	return uint256.FromHex(resp.Result)
}
