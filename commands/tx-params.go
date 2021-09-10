package commands

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"

	"github.com/holiman/uint256"
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
	To             *common.Address
	Value          *uint256.Int
	Data           []byte
	GasTip         *uint256.Int
	GasPrice       *uint256.Int
	Gas            *uint256.Int
	TxCount        *uint256.Int
	TxCountPending *uint256.Int
	Balance        *uint256.Int
}

type TransactionParamsOutput struct {
	RpcUrl         string `json:"rpcUrl"`
	ChainId        string `json:"chainId"`
	From           string `json:"from"`
	To             string `json:"to"`
	Value          string `json:"value"`
	Data           string `json:"data"`
	GasTip         string `json:"gasTip,omitempty"`
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
	fromAddr := common.BytesToAddress(hexutil.MustDecode(ctx.String(flags.FromParam.Name)))
	var toAddr *common.Address
	if ctx.IsSet(flags.ToParam.Name) {
		to := common.BytesToAddress(hexutil.MustDecode(ctx.String(flags.ToParam.Name)))
		toAddr = &to
	}
	if !ctx.IsSet(flags.DeployParam.Name) && toAddr == nil {
		return errors.New(fmt.Sprintf("Missing to address --%s", flags.ToParam.Name))
	}
	if ctx.IsSet(flags.DeployParam.Name) && toAddr != nil {
		return errors.New(fmt.Sprintf("to address --%s not accepted when we deploy", flags.ToParam.Name))
	}
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
	// https://docs.soliditylang.org/en/develop/abi-spec.html
	var data = []byte{}
	if ctx.IsSet(flags.DataParam.Name) {
		data = hexutil.MustDecode(ctx.String(flags.DataParam.Name))
	} else if ctx.IsSet(flags.MethodParam.Name) {
		errMsg := fmt.Sprintf("Method call needs to be specified in format (example): --%s=transfer:address,uint256", flags.MethodParam.Name)
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
		argTypes, packedValues, err := abiPackedValuesFromCli(ctx, typeNames)
		if err != nil {
			return err
		}
		method := NewHashedMethod(methodName, argTypes)
		data = append(method.Id[:], packedValues...)
	} else if ctx.IsSet(flags.DeployParam.Name) {
		var bin []byte
		if ctx.IsSet(flags.BinParam.Name) {
			var err error
			bin, err = hex.DecodeString(ctx.String(flags.BinParam.Name))
			if err != nil {
				return err
			}
		} else if ctx.IsSet(flags.BinFileParam.Name) {
			data, err := os.ReadFile(ctx.String(flags.BinFileParam.Name))
			if err != nil {
				return err
			}
			bin, err = hex.DecodeString(string(data))
			if err != nil {
				return err
			}
		} else {
			return errors.New(fmt.Sprintf("Missing contract binary (init code) --%s", flags.BinParam.Name))
		}
		var packedValues []byte
		typeNames := strings.Split(ctx.String(flags.DeployParam.Name), ",")
		if len(typeNames) > 0 {
			var err error
			_, packedValues, err = abiPackedValuesFromCli(ctx, typeNames)
			if err != nil {
				return err
			}
		}
		data = append(bin, packedValues...)
	}

	// either value or data needs to be specified
	if valbig == nil && len(data) == 0 {
		return errors.New(fmt.Sprintf("Either --%s, --%s, %s or %s needs to be specifed", flags.ValueParam.Name, flags.DataParam.Name, flags.MethodParam.Name, flags.DeployParam.Name))
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
		valueInGwei := new(uint256.Int).Div(p.Value, new(uint256.Int).SetUint64(params.GWei))
		term.Print(fmt.Sprintf("rpcUrl: %s", p.Endpoint))
		term.Print(fmt.Sprintf("chainId: %s", p.ChainId))
		term.Print(fmt.Sprintf("from: %s", p.From))
		term.Print(fmt.Sprintf("to: %s", p.To))
		term.Print(fmt.Sprintf("value: %s wei (%s gwei) (%.9f eth/ftm)", p.Value, valueInGwei, float64(valueInGwei.Uint64())/1e9))
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
	out := TransactionParamsOutput{
		RpcUrl:         p.Endpoint.Url(),
		ChainId:        p.ChainId.Hex(),
		From:           p.From.Hex(),
		Value:          p.Value.Hex(),
		Data:           hexutil.Encode(data),
		GasPrice:       p.GasPrice.Hex(),
		Gas:            p.Gas.Hex(),
		TxCount:        p.TxCount.Hex(),
		TxCountPending: p.TxCountPending.Hex(),
		Balance:        p.Balance.Hex(),
	}
	if p.To != nil {
		out.To = p.To.Hex()
	}
	if p.GasTip != nil && ctx.Bool(flags.NoTip.Name) == false {
		out.GasTip = p.GasTip.Hex()
	}
	b, err := json.Marshal(&out)
	if err != nil {
		return err
	}
	term.Output(fmt.Sprintf("%s\n", string(b)))
	return nil
}

func GetTransactionParams(term ui.Screen, endpoint rpc.RpcEndpoint, from common.Address, to *common.Address, value *uint256.Int, data []byte, tag BlockPositionTag) (*TransactionParams, error) {
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

func abiPackedValuesFromCli(ctx *cli.Context, typeNames []string) (abi.Arguments, []byte, error) {
	argTypes, err := AbiTypesFromStrings(typeNames)
	if err != nil {
		return nil, nil, err
	}
	argValues, err := AbiValuesFromCli(ctx, argTypes)
	if err != nil {
		return nil, nil, err
	}
	packedValues, err := AbiPackValues(argTypes, argValues)
	return argTypes, packedValues, err
}
