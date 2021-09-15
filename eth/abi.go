package eth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/holiman/uint256"
	"github.com/jaanek/jeth/abi"
	"github.com/jaanek/jeth/rpc"
	"github.com/jaanek/jeth/ui"
	"github.com/ledgerwatch/erigon/common"
	"github.com/urfave/cli"
)

const ReceiptWaitTime = 240 * time.Second

type TxSigner interface {
	GetSignedRawTx(chainID uint256.Int, nonce uint64, from common.Address, to *common.Address, value *uint256.Int, input []byte, gasLimit uint64, gasPrice, gasTip, gasFeeCap *uint256.Int) ([]byte, error)
}

func AbiPackValues(argTypes abi.Arguments, argValues []string) ([]byte, error) {
	values, err := AbiValuesFromTypes(argTypes, argValues)
	if err != nil {
		return nil, err
	}
	packedValues, err := argTypes.Pack(values...)
	if err != nil {
		return nil, err
	}
	return packedValues, nil
}

// parse provided method parameters against their types
func AbiValuesFromTypes(inputs abi.Arguments, values []string) ([]interface{}, error) {
	if len(values) != len(inputs) {
		return nil, errors.New("abi arg type's len != values len")
	}
	params := []interface{}{}
	for i, input := range inputs {
		arg := values[i]
		param, err := abi.ToGoTypeFromStr(input.Type, arg)
		if err != nil {
			return nil, err
		}
		params = append(params, param)
	}
	return params, nil
}

func AbiValuesFromCli(ctx *cli.Context, inputs abi.Arguments) ([]string, error) {
	values := []string{}
	for i := range inputs {
		argNum := strconv.FormatInt(int64(i), 10)
		if !ctx.IsSet(argNum) {
			return nil, fmt.Errorf("argument --%s not set", argNum)
		}
		arg := ctx.String(argNum)
		values = append(values, arg)
	}
	return values, nil
}

func Deploy(term ui.Screen, endpoint rpc.Endpoint, from common.Address, bin []byte, value *uint256.Int, typeNames []string, values []string, waitTime time.Duration, txSigner TxSigner) (string, *TxReceipt, error) {
	argTypes, err := abi.AbiTypesFromStrings(typeNames)
	if err != nil {
		return "", nil, err
	}
	packedValues, err := AbiPackValues(argTypes, values)
	if err != nil {
		return "", nil, err
	}
	data := append(bin, packedValues...)

	// estimate params, gas etc.
	params, err := GetTransactionParams(term, endpoint, from, nil, value, data, Latest)
	if err != nil {
		return "", nil, fmt.Errorf("Error while getting tx params for a method call: %w", err)
	}

	// get signed tx and send it
	encoded, err := txSigner.GetSignedRawTx(*params.ChainId, *params.TxCount, from, nil, value, data, *params.Gas, params.GasPrice, params.GasTip, params.GasPrice)
	hash, err := SendTransaction(term, endpoint, encoded)
	if err != nil {
		return "", nil, fmt.Errorf("Failed to send tx: %w", err)
	}
	// m.term.Print(fmt.Sprintf("Tx hash: %s", hash))

	// wait for tx receipt
	if waitTime < 0 {
		waitTime = ReceiptWaitTime
	}
	c, _ := context.WithTimeout(context.Background(), waitTime)
	receipt, err := WaitTransactionReceipt(c, term, endpoint, hash)
	if err != nil {
		return hash, nil, err
	}
	return hash, receipt, nil
}

func Send(term ui.Screen, endpoint rpc.Endpoint, from common.Address, to common.Address, value *uint256.Int, data []byte, waitTime time.Duration, txSigner TxSigner) (string, *TxReceipt, error) {
	// estimate params, gas etc.
	params, err := GetTransactionParams(term, endpoint, from, &to, value, data, Latest)
	if err != nil {
		return "", nil, fmt.Errorf("Error while getting tx params for a method call: %w", err)
	}

	// get signed tx and send it
	encoded, err := txSigner.GetSignedRawTx(*params.ChainId, *params.TxCount, from, &to, value, data, *params.Gas, params.GasPrice, params.GasTip, params.GasPrice)
	hash, err := SendTransaction(term, endpoint, encoded)
	if err != nil {
		return "", nil, fmt.Errorf("Failed to send tx: %w", err)
	}
	// m.term.Print(fmt.Sprintf("Tx hash: %s", hash))

	// wait for tx receipt
	if waitTime < 0 {
		waitTime = ReceiptWaitTime
	}
	c, _ := context.WithTimeout(context.Background(), waitTime)
	receipt, err := WaitTransactionReceipt(c, term, endpoint, hash)
	if err != nil {
		return hash, nil, err
	}
	return hash, receipt, nil
}
