package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/holiman/uint256"
	"github.com/jaanek/jeth/flags"
	"github.com/jaanek/jeth/rpc"
	"github.com/jaanek/jeth/ui"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/hexutil"
	"github.com/ledgerwatch/erigon/params"
	"github.com/urfave/cli"
)

func TransactionParamsCommand(term ui.Screen, ctx *cli.Context, endpoint rpc.RpcEndpoint) error {
	var wg sync.WaitGroup
	var errs = make(chan error, 6)
	var chainId, gasTip, gasPrice *uint256.Int
	var fromAddr *common.Address
	if ctx.IsSet(flags.FromParam.Name) {
		input := ctx.String(flags.FromParam.Name)
		data, err := hexutil.Decode(input)
		if err != nil {
			return err
		}
		from := common.BytesToAddress(data)
		fromAddr = &from
	}
	var countLatest *uint256.Int
	var countPending *uint256.Int
	var fromBalance *uint256.Int
	var fromBalanceSuffix string = "wei"

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
	if fromAddr != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			countLatest, err = TransactionsCount(term, endpoint, *fromAddr, Latest)
			if err != nil {
				errs <- err
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			countPending, err = TransactionsCount(term, endpoint, *fromAddr, Pending)
			if err != nil {
				errs <- err
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			fromBalance, err = GetAccountBalance(term, endpoint, *fromAddr)
			if err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()

	// check errors
	var err error
	select {
	case err = <-errs:
	default:
	}
	if err != nil {
		return err
	}

	// output results
	if ctx.IsSet(flags.Gwei.Name) {
		if gasTip != nil {
			gasTip = new(uint256.Int).Div(gasTip, new(uint256.Int).SetUint64(params.GWei))
		}
		gasPrice = new(uint256.Int).Div(gasPrice, new(uint256.Int).SetUint64(params.GWei))
	}
	if ctx.IsSet(flags.Eth.Name) {
		if fromBalance != nil {
			fromBalance = new(uint256.Int).Div(fromBalance, new(uint256.Int).SetUint64(params.Ether))
		}
		fromBalanceSuffix = "eth"
	}
	if ctx.IsSet(flags.Plain.Name) {
		term.Output(fmt.Sprintf("chainId: %s\n", chainId))
		term.Output(fmt.Sprintf("gasTip: %s\n", gasTip))
		term.Output(fmt.Sprintf("gasPrice: %s\n", gasPrice))
		if countLatest != nil {
			term.Output(fmt.Sprintf("txCountLatest: %s\n", countLatest))
		}
		if countPending != nil {
			term.Output(fmt.Sprintf("txCountPending: %s\n", countPending))
		}
		if fromBalance != nil {
			term.Output(fmt.Sprintf("balance: %v (%s)\n", fromBalance, fromBalanceSuffix))
		}
		return nil
	}
	type Output struct {
		ChainId        *uint256.Int `json:"chainId"`
		GasTip         *uint256.Int `json:"gasTip"`
		GasPrice       *uint256.Int `json:"gasPrice"`
		TxCountLatest  *uint256.Int `json:"txCountLatest"`
		TxCountPending *uint256.Int `json:"txCountPending"`
		Balance        *uint256.Int `json:"balance"`
	}
	output := Output{
		ChainId:        chainId,
		GasTip:         gasTip,
		GasPrice:       gasPrice,
		TxCountLatest:  countLatest,
		TxCountPending: countPending,
		Balance:        fromBalance,
	}
	bytes, err := json.Marshal(&output)
	if err != nil {
		return err
	}
	term.Output(fmt.Sprintf("%s\n", string(bytes)))
	return nil
}
