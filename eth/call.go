package eth

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/holiman/uint256"
	"github.com/jaanek/jeth/flags"
	"github.com/jaanek/jeth/httpclient"
	"github.com/jaanek/jeth/rpc"
	"github.com/jaanek/jeth/ui"
	"github.com/ledgerwatch/erigon/accounts/abi"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/hexutil"
	"github.com/ledgerwatch/erigon/common/math"
	"github.com/ledgerwatch/erigon/params"
	"github.com/urfave/cli"
)

type CallMethodParam struct {
	From     string  `json:"from,omitempty"`
	To       string  `json:"to"`
	Value    string  `json:"value,omitempty"`
	Data     string  `json:"data"`
	Gas      *string `json:"gas,omitempty"`
	GasPrice *string `json:"gasPrice,omitempty"`
}

type CallOutput struct {
	Result          string               `json:"result"`
	UnpackedResults []CallUnpackedResult `json:"unpacked"`
}

type CallUnpackedResult struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

func (r CallUnpackedResult) ToUint256() (*uint256.Int, error) {
	if r.Type != "uint256" {
		return nil, fmt.Errorf("Value type not uint256. Type: %v, value: %v: ", r.Type, r.Value)
	}
	big, ok := r.Value.(*big.Int)
	if !ok {
		return nil, fmt.Errorf("Value not *big.Int. Type: %v, value: %v: ", r.Type, r.Value)
	}
	val, overflow := uint256.FromBig(big)
	if overflow {
		return nil, fmt.Errorf("*big.Int to *uint256.Int overflow error. Type: %v, value: %v: ", r.Type, r.Value)
	}
	return val, nil
}

func CallMethodCommand(term ui.Screen, ctx *cli.Context, endpoint rpc.Endpoint) error {
	// validate args
	var fromAddr *common.Address
	if ctx.IsSet(flags.FromParam.Name) {
		addr := common.BytesToAddress(hexutil.MustDecode(ctx.String(flags.FromParam.Name)))
		fromAddr = &addr
	}
	if !ctx.IsSet(flags.ToParam.Name) {
		return errors.New(fmt.Sprintf("Missing to address --%s", flags.ToParam.Name))
	}
	toAddr := common.BytesToAddress(hexutil.MustDecode(ctx.String(flags.ToParam.Name)))

	var value *uint256.Int
	if ctx.IsSet(flags.ValueParam.Name) {
		var valbig *big.Int
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
		value := new(uint256.Int)
		value.SetFromBig(valbig)
	}

	if !ctx.IsSet(flags.MethodParam.Name) {
		return errors.New(fmt.Sprintf("Missing method param --%s", flags.MethodParam.Name))
	}
	errMsg := fmt.Sprintf("Method call needs to be specified in format (example): --%s=transfer:address,uint256", flags.MethodParam.Name)
	methodStr := ctx.String(flags.MethodParam.Name)
	methodSplit := strings.Split(methodStr, ":")
	if len(methodSplit) != 2 {
		return errors.New(errMsg)
	}
	methodName := methodSplit[0]
	var packedValues []byte
	typeNames := strings.Split(methodSplit[1], ",")
	argTypes, packedValues, err := abiPackedValuesFromCli(ctx, typeNames)
	if err != nil {
		return err
	}
	method := NewHashedMethod(methodName, argTypes)
	data := append(method.Id[:], packedValues...)

	// call
	result, err := CallMethod(term, endpoint, fromAddr, toAddr, value, data, Latest)
	if err != nil {
		return err
	}
	out := CallOutput{
		Result: hexutil.Encode(result),
	}
	if ctx.IsSet(flags.OutputTypesParam.Name) {
		typeNames := strings.Split(ctx.String(flags.OutputTypesParam.Name), ",")
		outTypes, err := AbiTypesFromStrings(typeNames)
		if err != nil {
			return err
		}
		results, err := UnpackCallResult(result, outTypes)
		if err != nil {
			term.Print(fmt.Sprintf("Could not unpack output param! Error: %v", err))
		}
		out.UnpackedResults = results
	}
	if ctx.IsSet(flags.Plain.Name) {
		term.Print(fmt.Sprintf("result: %s", out.Result))
		term.Print(fmt.Sprintf("unpacked results: %+v", out.UnpackedResults))
	}
	b, err := json.Marshal(&out)
	if err != nil {
		return err
	}
	term.Output(fmt.Sprintf("%s\n", string(b)))
	return nil
}

func CallMethod(term ui.Screen, endpoint rpc.Endpoint, from *common.Address, to common.Address, value *uint256.Int, data []byte, tag BlockPositionTag) ([]byte, error) {
	param := CallMethodParam{
		To:   to.Hex(),
		Data: hexutil.Encode(data),
	}
	if from != nil {
		param.From = from.Hex()
	}
	if value != nil {
		param.Value = value.Hex()
	}
	client := httpclient.NewDefault(term)
	resp := rpc.RpcResultStr{}
	err := rpc.Call(term, client, endpoint, "eth_call", []interface{}{param, tag}, &resp)
	if err != nil {
		return nil, err
	}
	return hexutil.Decode(resp.Result)
}

func UnpackCallResult(result []byte, outTypes abi.Arguments) ([]CallUnpackedResult, error) {
	unpackedResults, err := outTypes.Unpack(result)
	if err != nil {
		return nil, err
	} else {
		results := []CallUnpackedResult{}
		for i, r := range unpackedResults {
			results = append(results, CallUnpackedResult{
				Type:  outTypes[i].Type.String(),
				Value: r,
			})
		}
		return results, nil
	}
}
