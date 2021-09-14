package eth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/holiman/uint256"
	"github.com/jaanek/jeth/abipack"
	"github.com/jaanek/jeth/rpc"
	"github.com/jaanek/jeth/ui"
	"github.com/ledgerwatch/erigon/accounts/abi"
	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/crypto"
	"github.com/urfave/cli"
)

const ReceiptWaitTime = 240 * time.Second

type HashedMethod struct {
	Sig string
	Id  [4]byte
}

type TxSigner interface {
	GetSignedRawTx(chainID uint256.Int, nonce uint64, from common.Address, to *common.Address, value *uint256.Int, input []byte, gasLimit uint64, gasPrice, gasTip, gasFeeCap *uint256.Int) ([]byte, error)
}

func AbiPackedMethodCall(methodName string, types []string, values []string) ([]byte, error) {
	argTypes, err := AbiTypesFromStrings(types)
	if err != nil {
		return nil, err
	}
	packedValues, err := AbiPackValues(argTypes, values)
	if err != nil {
		return nil, err
	}
	method := NewHashedMethod(methodName, argTypes)
	return append(method.Id[:], packedValues...), nil
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

func AbiTypesFromStrings(typeNames []string) (abi.Arguments, error) {
	var argTypes = make([]abi.Argument, 0, len(typeNames))
	for _, argTypeName := range typeNames {
		if len(argTypeName) == 0 {
			continue
		}
		argType, err := abi.NewType(argTypeName, "", nil) // example: "uint256"
		if err != nil {
			return nil, fmt.Errorf("argument contains invalid type: %s. Error: %w", argTypeName, err)
		}
		arg := abi.Argument{
			Name:    "",
			Type:    argType,
			Indexed: false,
		}
		argTypes = append(argTypes, arg)
	}
	return argTypes, nil
}

// parse provided method parameters against their types
func AbiValuesFromTypes(inputs abi.Arguments, values []string) ([]interface{}, error) {
	if len(values) != len(inputs) {
		return nil, errors.New("abi arg type's len != values len")
	}
	params := []interface{}{}
	for i, input := range inputs {
		arg := values[i]
		param, err := abipack.ToGoType(input.Type, arg)
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
	Call(from *common.Address, to common.Address, value *uint256.Int, values []string) ([]CallUnpackedResult, error)
}

func NewMethod(term ui.Screen, endpoint rpc.Endpoint, methodName string, inputs []string, outputs []string) (Method, error) {
	argTypes, err := AbiTypesFromStrings(inputs)
	if err != nil {
		return nil, err
	}
	outTypes, err := AbiTypesFromStrings(outputs)
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
	packedValues, err := AbiPackValues(m.inputs, values)
	if err != nil {
		return nil, err
	}
	return append(m.method.Id[:], packedValues...), nil
}

func (m *method) UnpackResult(result []byte) ([]CallUnpackedResult, error) {
	return UnpackCallResult(result, m.outputs)
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

func (m *method) Call(from *common.Address, to common.Address, value *uint256.Int, values []string) ([]CallUnpackedResult, error) {
	data, err := m.PackedCall(values)
	if err != nil {
		return nil, fmt.Errorf("Error while packing method call: %w", err)
	}
	result, err := CallMethod(m.term, m.endpoint, from, to, value, data, Latest)
	if err != nil {
		return nil, err
	}
	results, err := UnpackCallResult(result, m.outputs)
	if err != nil {
		m.term.Print(fmt.Sprintf("Could not unpack output param! Error: %v", err))
	}
	return results, nil
}

func Deploy(term ui.Screen, endpoint rpc.Endpoint, from common.Address, bin []byte, value *uint256.Int, typeNames []string, values []string, waitTime time.Duration, txSigner TxSigner) (string, *TxReceipt, error) {
	argTypes, err := AbiTypesFromStrings(typeNames)
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
