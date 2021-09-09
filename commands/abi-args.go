package commands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jaanek/jeth/abipack"
	"github.com/ledgerwatch/erigon/accounts/abi"
	"github.com/ledgerwatch/erigon/crypto"
	"github.com/urfave/cli"
)

type HashedMethod struct {
	Sig string
	Id  [4]byte
}

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
		return nil, errors.New("abi input's len != values len")
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
