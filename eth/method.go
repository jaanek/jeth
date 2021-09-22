package eth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/holiman/uint256"
	"github.com/jaanek/jeth/abi"
	"github.com/jaanek/jeth/rpc"
	"github.com/jaanek/jeth/ui"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/crypto"
)

type MethodSpec struct {
	Name    string
	Inputs  []string
	Outputs []string
}

func AbiPackedMethodCall(methodName string, types []string, values []string) ([]byte, error) {
	argTypes, err := abi.TypesFromStrings(types)
	if err != nil {
		return nil, err
	}
	packedValues, err := abi.PackValues(argTypes, values)
	if err != nil {
		return nil, err
	}
	method := NewHashedMethod(methodName, argTypes)
	return append(method.Id[:], packedValues...), nil
}

type HashedMethod struct {
	Sig string
	Id  [4]byte
}

// https://docs.soliditylang.org/en/develop/abi-spec.html
func NewHashedMethod(methodName string, argTypes abi.Arguments) HashedMethod {
	var types = make([]string, len(argTypes))
	for i, input := range argTypes {
		types[i] = input.Type.String()
	}
	method := HashedMethod{}
	method.Sig = fmt.Sprintf("%v(%v)", methodName, strings.Join(types, ","))
	sig := crypto.Keccak256([]byte(method.Sig))[:4]
	copy(method.Id[:], sig)
	return method
}

type method struct {
	term       ui.Screen
	endpoint   rpc.Endpoint
	methodName string
	method     HashedMethod
	inputs     abi.Arguments
	outputs    abi.Arguments
}

func (m *method) Name() string {
	return m.methodName
}

func (m *method) Inputs() abi.Arguments {
	return m.inputs
}

func (m *method) Outputs() abi.Arguments {
	return m.outputs
}

type Method interface {
	Name() string
	Inputs() abi.Arguments
	Outputs() abi.Arguments
	Send(from common.Address, to common.Address, value *uint256.Int, values []string, waitTime time.Duration, txSigner TxSigner) (string, *TxReceipt, error)
	Call(from *common.Address, to common.Address, value *uint256.Int, values []string) ([]byte, []abi.UnpackedValue, error)
}

func NewMethod(term ui.Screen, endpoint rpc.Endpoint, methodName string, inputs []string, outputs []string) (Method, error) {
	argTypes, err := abi.TypesFromStrings(inputs)
	if err != nil {
		return nil, err
	}
	outTypes, err := abi.TypesFromStrings(outputs)
	if err != nil {
		return nil, err
	}
	hashedMethod := NewHashedMethod(methodName, argTypes)
	return &method{
		term:       term,
		endpoint:   endpoint,
		methodName: methodName,
		method:     hashedMethod,
		inputs:     argTypes,
		outputs:    outTypes,
	}, nil
}

func (m *method) PackedCall(values []string) ([]byte, error) {
	packedValues, err := abi.PackValues(m.inputs, values)
	if err != nil {
		return nil, err
	}
	return append(m.method.Id[:], packedValues...), nil
}

func (m *method) UnpackResult(result []byte) ([]abi.UnpackedValue, error) {
	return abi.UnpackAbiData(m.outputs, result)
}

type GetSignedTxCallback = func(term ui.Screen, chainID uint256.Int, nonce uint64, from common.Address, to *common.Address, value *uint256.Int, input []byte, gasLimit uint64, gasPrice, gasTip, gasFeeCap *uint256.Int) ([]byte, error)

func (m *method) Send(from common.Address, to common.Address, value *uint256.Int, values []string, waitTime time.Duration, txSigner TxSigner) (string, *TxReceipt, error) {
	data, err := m.PackedCall(values)
	if err != nil {
		return "", nil, fmt.Errorf("Error while packing method call: %w", err)
	}

	// estimate params, gas etc.
	params, err := GetTransactionParams(m.term, m.endpoint, from, &to, value, data, Latest)
	if err != nil {
		return "", nil, fmt.Errorf("Error while getting tx params for a method call: %w", err)
	}

	// get signed tx and send it
	encoded, err := txSigner.GetSignedRawTx(*params.ChainId, *params.TxCount, from, &to, value, data, *params.Gas, params.GasPrice, params.GasTip, params.GasPrice)
	hash, err := SendTransaction(m.term, m.endpoint, encoded)
	if err != nil {
		return "", nil, fmt.Errorf("Failed to send tx: %w", err)
	}
	// m.term.Print(fmt.Sprintf("Tx hash: %s", hash))

	// wait for tx receipt
	if waitTime < 0 {
		waitTime = ReceiptWaitTime
	}
	c, _ := context.WithTimeout(context.Background(), waitTime)
	receipt, err := WaitTransactionReceipt(c, m.term, m.endpoint, hash)
	if err != nil {
		return hash, nil, err
	}
	return hash, receipt, nil
}

func (m *method) Call(from *common.Address, to common.Address, value *uint256.Int, values []string) ([]byte, []abi.UnpackedValue, error) {
	data, err := m.PackedCall(values)
	if err != nil {
		return nil, nil, fmt.Errorf("Error while packing method call: %w", err)
	}
	result, err := CallMethod(m.term, m.endpoint, from, to, value, data, Latest)
	if err != nil {
		return nil, nil, err
	}
	results, err := abi.UnpackAbiData(m.outputs, result)
	if err != nil {
		m.term.Print(fmt.Sprintf("Could not unpack output param! Error: %v", err))
	}
	return result, results, nil
}
