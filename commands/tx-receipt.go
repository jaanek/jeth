package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/holiman/uint256"
	"github.com/jaanek/jeth/flags"
	"github.com/jaanek/jeth/httpclient"
	"github.com/jaanek/jeth/rpc"
	"github.com/jaanek/jeth/ui"
	"github.com/urfave/cli"
)

type RpcResultTxReceipt struct {
	rpc.RpcResultStr
	Result *TxReceipt `json:"result"`
}

type TxReceipt struct {
	BlockHash         string  `json:"blockHash"`
	BlockNumber       string  `json:"blockNumber"`
	ContractAddress   string  `json:"contractAddress"`
	CumulativeGasUsed string  `json:"cumulativeGasUsed"`
	EffectiveGasUsed  string  `json:"effectiveGasPrice"`
	From              string  `json:"from"`
	GasUsed           string  `json:"gasUsed"`
	Logs              []TxLog `json:"logs"`
	LogsBloom         string  `json:"logsBloom"`
	Status            string  `json:"status"`
	To                string  `json:"to"`
	TransactionHash   string  `json:"transactionHash"`
	TransactionIndex  string  `json:"transactionIndex"`
	Type              string  `json:"type"`
}

type TxLog struct {
	Address          string   `json:"address"`
	Topics           []string `json:"topics"`
	Data             string   `json:"data"`
	BlockNumber      string   `json:"blockNumber"`
	TransactionHash  string   `json:"transactionHash"`
	TransactionIndex string   `json:"transactionIndex"`
	BlockHash        string   `json:"blockHash"`
	LogIndex         string   `json:"logIndex"`
	Removed          bool     `json:"removed"`
}

func GetTransactionReceiptCommand(term ui.Screen, ctx *cli.Context, endpoint rpc.RpcEndpoint) error {
	// validate args
	if !ctx.IsSet(flags.HexParam.Name) {
		return errors.New(fmt.Sprintf("Missing tx hash --%s", flags.HexParam.Name))
	}
	input := ctx.String(flags.HexParam.Name)
	if !(strings.HasPrefix(input, "0x") || strings.HasPrefix(input, "0X")) {
		return errors.New("Tx hash needs to start with 0x")
	}

	// call
	receipt, err := GetTransactionReceipt(term, endpoint, input)
	if err != nil {
		return err
	}
	term.Output(fmt.Sprintf("%+v\n", receipt))
	return nil
}

// returns tx receipt
func GetTransactionReceipt(term ui.Screen, endpoint rpc.RpcEndpoint, txHash string) (*TxReceipt, error) {
	client := httpclient.NewDefault(term)
	resp := RpcResultTxReceipt{}
	err := rpc.Call(term, client, endpoint, "eth_getTransactionReceipt", StringsToInterfaces([]string{txHash}), &resp)
	if err != nil {
		return nil, err
	}
	return resp.Result, nil
}

func WaitTransactionReceipt(ctx context.Context, term ui.Screen, endpoint rpc.RpcEndpoint, txHash string) (*TxReceipt, error) {
	waitTicker := time.NewTicker(time.Second)
	defer waitTicker.Stop()

	logEvery := time.NewTicker(5 * time.Second)
	defer logEvery.Stop()

	var blockNumber *uint256.Int
	for {
		receipt, err := GetTransactionReceipt(term, endpoint, txHash)
		if err != nil {
			return nil, err
		}
		if receipt != nil {
			return receipt, nil
		}
		blockNumber, _ = BlockNumber(term, endpoint)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-logEvery.C:
			if blockNumber != nil {
				term.Print(fmt.Sprintf("block number: %s", blockNumber))
			}
		case <-waitTicker.C:
		}
	}
}
