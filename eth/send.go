package eth

import (
	"context"
	"fmt"
	"time"

	"github.com/holiman/uint256"
	"github.com/jaanek/jeth/abi"
	"github.com/jaanek/jeth/rpc"
	"github.com/jaanek/jeth/ui"
	"github.com/ledgerwatch/erigon/common"
)

const ReceiptWaitTime = 240 * time.Second

type TxSigner interface {
	GetSignedRawTx(chainID uint256.Int, nonce uint64, from common.Address, to *common.Address, value *uint256.Int, input []byte, gasLimit uint64, gasPrice, gasTip, gasFeeCap *uint256.Int) ([]byte, error)
}

func Deploy(term ui.Screen, endpoint rpc.Endpoint, from common.Address, bin []byte, value *uint256.Int, typeNames []string, values []string, waitTime time.Duration, txSigner TxSigner) (string, *TxReceipt, error) {
	argTypes, err := abi.TypesFromStrings(typeNames)
	if err != nil {
		return "", nil, err
	}
	packedValues, err := abi.PackValues(argTypes, values)
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

func SendValue(term ui.Screen, endpoint rpc.Endpoint, from common.Address, to common.Address, value *uint256.Int, waitTime time.Duration, txSigner TxSigner) (string, *TxReceipt, error) {
	return Send(term, endpoint, from, to, value, []byte{}, waitTime, txSigner)
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
