package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"

	"github.com/holiman/uint256"
	"github.com/jaanek/jeth/abipack"
	"github.com/jaanek/jeth/flags"
	"github.com/jaanek/jeth/rpc"
	"github.com/jaanek/jeth/ui"
	"github.com/ledgerwatch/erigon/accounts/abi"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/hexutil"
	"github.com/ledgerwatch/erigon/common/math"
	"github.com/ledgerwatch/erigon/params"
	"github.com/urfave/cli"
)

type TransactionParams struct {
	Endpoint       rpc.RpcEndpoint
	ChainId        *uint256.Int
	From           common.Address
	To             common.Address
	Value          *uint256.Int
	Data           []byte
	GasTip         *uint256.Int
	GasPrice       *uint256.Int
	Gas            *uint256.Int
	TxCount        *uint256.Int
	TxCountPending *uint256.Int
	Balance        *uint256.Int
}

type Output struct {
	RpcUrl         string `json:"rpcUrl"`
	ChainId        string `json:"chainId"`
	From           string `json:"from"`
	To             string `json:"to"`
	Value          string `json:"value"`
	Data           string `json:"data"`
	GasTip         string `json:"gasTip"`
	GasPrice       string `json:"gasPrice"`
	Gas            string `json:"gas"`
	TxCount        string `json:"txCount"`
	TxCountPending string `json:"txCountPending"`
	Balance        string `json:"balance"`
}

func TransactionParamsCommand(term ui.Screen, ctx *cli.Context, endpoint rpc.RpcEndpoint) error {
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
	}

	// check data or method call
	var data = []byte{}
	if ctx.IsSet(flags.DataParam.Name) {
		data = hexutil.MustDecode(ctx.String(flags.DataParam.Name))
	} else if ctx.IsSet(flags.MethodParam.Name) {
		errMsg := fmt.Sprintf("Method call needs to be specified in format (example): --%s=transfer:uint256,address", flags.MethodParam.Name)
		methodStr := ctx.String(flags.MethodParam.Name)
		methodSplit := strings.Split(methodStr, ":")
		if len(methodSplit) != 2 {
			return errors.New(errMsg)
		}
		methodName := methodSplit[0]
		typeNames := strings.Split(methodSplit[1], ",")
		if len(typeNames) == 0 {
			return errors.New(errMsg)
		}
		var argTypes = make([]abi.Argument, 0, len(typeNames))
		for _, argTypeName := range typeNames {
			argType, err := abi.NewType(argTypeName, "", nil) // example: "uint256"
			if err != nil {
				return fmt.Errorf("method --%s contains invalid type: %s. Error: %w", flags.MethodParam.Name, argTypeName, err)
			}
			arg := abi.Argument{
				Name:    "amount",
				Type:    argType,
				Indexed: false,
			}
			argTypes = append(argTypes, arg)
		}
		method := abi.NewMethod(methodName, methodName, abi.Function, "", false, false, argTypes, nil)
		// parse provided method parameters against their types
		params := []interface{}{}
		for i, input := range method.Inputs {
			paramName := strconv.FormatInt(int64(i), 10)
			if !ctx.IsSet(paramName) {
				return fmt.Errorf("method argument --%s not set", paramName)
			}
			arg := ctx.String(paramName)
			param, err := abipack.ToGoType(input.Type, arg)
			if err != nil {
				return err
			}
			params = append(params, param)
		}
		packedArgs, err := method.Inputs.Pack(params...)
		if err != nil {
			return err
		}
		packed, err := append(method.ID, packedArgs...), nil
		if err != nil {
			return err
		}
		data = packed
		term.Print(fmt.Sprintf("packed data: %x", data))
	}

	// either value or data needs to be specified
	if valbig == nil && len(data) == 0 {
		return errors.New(fmt.Sprintf("Either --%s, --%s or %s needs to be specifed", flags.ValueParam.Name, flags.DataParam.Name, flags.MethodParam.Name))
	}
	value := new(uint256.Int)
	if valbig != nil {
		value.SetFromBig(valbig)
	}

	// call
	p, err := GetTransactionParams(term, endpoint, fromAddr, toAddr, value, data, Latest)
	if err != nil {
		return err
	}

	// output results
	if ctx.IsSet(flags.Plain.Name) {
		term.Print(fmt.Sprintf("rpcUrl: %s", p.Endpoint))
		term.Print(fmt.Sprintf("chainId: %s", p.ChainId))
		term.Print(fmt.Sprintf("from: %s", p.From))
		term.Print(fmt.Sprintf("to: %s", p.To))
		term.Print(fmt.Sprintf("value: %s wei", p.Value))
		term.Print(fmt.Sprintf("data: %x", p.Data))
		if p.GasTip != nil {
			gasTipInGwei := new(uint256.Int).Div(p.GasTip, new(uint256.Int).SetUint64(params.GWei))
			term.Print(fmt.Sprintf("gasTip: %s wei (%s gwei)", p.GasTip, gasTipInGwei))
		}
		if p.GasPrice != nil {
			gasPriceInGwei := new(uint256.Int).Div(p.GasPrice, new(uint256.Int).SetUint64(params.GWei))
			term.Print(fmt.Sprintf("gasPrice: %s wei (%s gwei)", p.GasPrice, gasPriceInGwei))
		}
		term.Print(fmt.Sprintf("gas: %s", p.Gas))
		if p.TxCount != nil {
			term.Print(fmt.Sprintf("txCountLatest: %s", p.TxCount))
		}
		if p.TxCountPending != nil {
			term.Print(fmt.Sprintf("txCountPending: %s", p.TxCountPending))
		}
		if p.Balance != nil {
			balanceInEth := new(uint256.Int).Div(p.Balance, new(uint256.Int).SetUint64(params.Ether))
			term.Print(fmt.Sprintf("balance: %v (%s eth/ftm or chain native currency)", p.Balance, balanceInEth))
		}
	}
	out := Output{
		RpcUrl:         p.Endpoint.Url(),
		ChainId:        p.ChainId.Hex(),
		From:           p.From.Hex(),
		To:             p.To.Hex(),
		Value:          p.Value.Hex(),
		Data:           hexutil.Encode(data),
		GasPrice:       p.GasPrice.Hex(),
		Gas:            p.Gas.Hex(),
		TxCount:        p.TxCount.Hex(),
		TxCountPending: p.TxCountPending.Hex(),
		Balance:        p.Balance.Hex(),
	}
	if p.GasTip != nil {
		out.GasTip = p.GasTip.Hex()
	}
	b, err := json.Marshal(&out)
	if err != nil {
		return err
	}
	term.Output(fmt.Sprintf("%s\n", string(b)))
	return nil
}

func GetTransactionParams(term ui.Screen, endpoint rpc.RpcEndpoint, from common.Address, to common.Address, value *uint256.Int, data []byte, tag BlockPositionTag) (*TransactionParams, error) {
	var wg sync.WaitGroup
	var errs = make(chan error, 7)
	var chainId, gasTip, gasPrice, gas *uint256.Int
	var txCount *uint256.Int
	var txCountPending *uint256.Int
	var fromBalance *uint256.Int

	// trigger the rpc
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		gasTip, err = MaxPriorityFeePerGas(term, endpoint)
		if err != nil {
			var e *rpc.RpcError
			if errors.As(err, &e) && strings.Contains(e.Message, "does not exist") {
				gasTip = nil
				return
			}
			errs <- fmt.Errorf("failed to retrieve maxPriorityFeePerGas: %w", err)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		gasPrice, err = GasPrice(term, endpoint)
		if err != nil {
			errs <- fmt.Errorf("failed to retrieve gasPrice: %w", err)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		chainId, err = ChainId(term, endpoint)
		if err != nil {
			errs <- fmt.Errorf("failed to retrieve chainId: %w", err)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		txCount, err = TransactionsCount(term, endpoint, from, tag)
		if err != nil {
			errs <- fmt.Errorf("failed to retrieve transaction count: %w", err)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		txCountPending, err = TransactionsCount(term, endpoint, from, Pending)
		if err != nil {
			errs <- fmt.Errorf("failed to retrieve transaction count: %w", err)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		fromBalance, err = GetAccountBalance(term, endpoint, from)
		if err != nil {
			errs <- fmt.Errorf("failed to retrieve account balance: %w", err)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		gas, err = EstimateGas(term, endpoint, from, to, value, data, tag)
		if err != nil {
			errs <- fmt.Errorf("failed to estimate gas: %w", err)
		}
	}()
	wg.Wait()

	// check errors
	var err error
	select {
	case err = <-errs:
	default:
	}
	if err != nil {
		return nil, err
	}
	return &TransactionParams{
		Endpoint:       endpoint,
		ChainId:        chainId,
		From:           from,
		To:             to,
		Value:          value,
		Data:           data,
		GasTip:         gasTip,
		GasPrice:       gasPrice,
		Gas:            gas,
		TxCount:        txCount,
		TxCountPending: txCountPending,
		Balance:        fromBalance,
	}, nil
}
