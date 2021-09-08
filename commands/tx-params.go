package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/holiman/uint256"
	"github.com/jaanek/jeth/flags"
	"github.com/jaanek/jeth/rpc"
	"github.com/jaanek/jeth/ui"
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
	Input          []byte
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
	Input          string `json:"input"`
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
	var input = []byte{}
	if ctx.IsSet(flags.InputParam.Name) {
		input = hexutil.MustDecode(ctx.String(flags.InputParam.Name))
	}
	// either value or input needs to be specified
	if valbig == nil && len(input) == 0 {
		return errors.New(fmt.Sprintf("Either --%s or --%s needs to be specifed", flags.ValueParam.Name, flags.InputParam.Name))
	}
	value := new(uint256.Int)
	if valbig != nil {
		value.SetFromBig(valbig)
	}

	// call
	p, err := GetTransactionParams(term, endpoint, fromAddr, toAddr, value, input, Latest)

	// output results
	if ctx.IsSet(flags.Plain.Name) {
		term.Print(fmt.Sprintf("rpcUrl: %s", p.Endpoint))
		term.Print(fmt.Sprintf("chainId: %s", p.ChainId))
		term.Print(fmt.Sprintf("from: %s", p.From))
		term.Print(fmt.Sprintf("to: %s", p.To))
		term.Print(fmt.Sprintf("value: %s wei", p.Value))
		term.Print(fmt.Sprintf("input: %s", p.Input))
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
		Input:          hexutil.Encode(input),
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

func GetTransactionParams(term ui.Screen, endpoint rpc.RpcEndpoint, from common.Address, to common.Address, value *uint256.Int, input []byte, tag BlockPositionTag) (*TransactionParams, error) {
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
			errs <- err
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		gasPrice, err = GasPrice(term, endpoint)
		if err != nil {
			errs <- err
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		chainId, err = ChainId(term, endpoint)
		if err != nil {
			errs <- err
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		txCount, err = TransactionsCount(term, endpoint, from, tag)
		if err != nil {
			errs <- err
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		txCountPending, err = TransactionsCount(term, endpoint, from, Pending)
		if err != nil {
			errs <- err
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		fromBalance, err = GetAccountBalance(term, endpoint, from)
		if err != nil {
			errs <- err
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		gas, err = EstimateGas(term, endpoint, from, to, value, input, tag)
		if err != nil {
			errs <- err
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
		Input:          input,
		GasTip:         gasTip,
		GasPrice:       gasPrice,
		Gas:            gas,
		TxCount:        txCount,
		TxCountPending: txCountPending,
		Balance:        fromBalance,
	}, nil
}
