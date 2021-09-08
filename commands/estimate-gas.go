package commands

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/holiman/uint256"
	"github.com/jaanek/jeth/flags"
	"github.com/jaanek/jeth/httpclient"
	"github.com/jaanek/jeth/rpc"
	"github.com/jaanek/jeth/ui"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/hexutil"
	"github.com/ledgerwatch/erigon/common/math"
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
	// validate args
	if !ctx.IsSet(flags.FromParam.Name) {
		return errors.New(fmt.Sprintf("Missing from address --%s", flags.FromParam.Name))
	}
	if !ctx.IsSet(flags.ToParam.Name) {
		return errors.New(fmt.Sprintf("Missing to address --%s", flags.ToParam.Name))
	}
	fromAddr := common.BytesToAddress(hexutil.MustDecode(ctx.String(flags.FromParam.Name)))
	toAddr := common.BytesToAddress(hexutil.MustDecode(ctx.String(flags.ToParam.Name)))
	var valbig *big.Int
	if ctx.IsSet(flags.ValueParam.Name) {
		var ok bool
		valbig, ok = math.ParseBig256(ctx.String(flags.ValueParam.Name))
		if !ok {
			return errors.New(fmt.Sprintf("invalid 256 bit integer: " + ctx.String(flags.ValueParam.Name)))
		}
		if ctx.IsSet(flags.ValueInEthParam.Name) {
			valbig = new(big.Int).Mul(valbig, new(big.Int).SetInt64(params.Ether))
		} else if ctx.IsSet(flags.ValueInGweiParam.Name) {
			valbig = new(big.Int).Mul(valbig, new(big.Int).SetInt64(params.GWei))
		}
	} else {
		valbig = new(big.Int)
	}
	var data = []byte{}
	if ctx.IsSet(flags.DataParam.Name) {
		data = hexutil.MustDecode(ctx.String(flags.DataParam.Name))
	}
	// either value or data needs to be specified
	if valbig.Cmp(new(big.Int)) == 0 && len(data) == 0 {
		return errors.New(fmt.Sprintf("Either --%s or --%s needs to be specifed", flags.ValueParam.Name, flags.DataParam.Name))
	}
	value := new(uint256.Int)
	value.SetFromBig(valbig)

	// call
	gas, err := EstimateGas(term, endpoint, fromAddr, toAddr, value, data, "latest")
	if err != nil {
		return err
	}
	term.Output(fmt.Sprintf("%s\n", gas))
	return nil
}

func EstimateGas(term ui.Screen, endpoint rpc.RpcEndpoint, from common.Address, to common.Address, value *uint256.Int, data []byte, tag BlockPositionTag) (*uint256.Int, error) {
	client := httpclient.NewDefault(term)
	resp := rpc.RpcResultStr{}
	err := rpc.Call(term, client, endpoint, "eth_estimateGas", []interface{}{EstimateGasParam{
		From:  from.Hex(),
		To:    to.Hex(),
		Value: value.Hex(),
		Data:  hexutil.Encode(data),
	}, tag}, &resp)
	if err != nil {
		return nil, err
	}
	return uint256.FromHex(resp.Result)
}

func DecodeTransaction(input string) (types.Transaction, error) {
	data, err := hexutil.Decode(input)
	if err != nil {
		return nil, err
	}
	tx, err := types.DecodeTransaction(rlp.NewStream(bytes.NewReader(data), uint64(len(data))))
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func GetSigner(tx types.Transaction) (common.Address, error) {
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
	return tx.Sender(*signer)
}
