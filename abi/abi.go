package abi

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"

	"github.com/holiman/uint256"
	"github.com/urfave/cli"
)

// abi, err := abi.JSON(strings.NewReader(abis[i]))

type UnpackedValue struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

func (r UnpackedValue) ToUint256() (*uint256.Int, error) {
	if r.Type != "uint256" {
		return nil, fmt.Errorf("Value type not uint256. Type: %v, value: %v: ", r.Type, r.Value)
	}
	big, ok := r.Value.(*big.Int)
	if !ok {
		return nil, fmt.Errorf("Value not *big.Int. Type: %v, value: %v: ", r.Type, r.Value)
	}
	val, overflow := uint256.FromBig(big)
	if overflow {
		return nil, fmt.Errorf("*big.Int to *uint256.Int overflow error. Type: %v, value: %v: ", r.Type, r.Value)
	}
	return val, nil
}

func PackValues(argTypes Arguments, argValues []string) ([]byte, error) {
	values, err := ValuesFromTypes(argTypes, argValues)
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
func ValuesFromTypes(inputs Arguments, values []string) ([]interface{}, error) {
	if len(values) != len(inputs) {
		return nil, errors.New("abi arg type's len != values len")
	}
	params := []interface{}{}
	for i, input := range inputs {
		arg := values[i]
		param, err := ToGoTypeFromStr(input.Type, arg)
		if err != nil {
			return nil, err
		}
		params = append(params, param)
	}
	return params, nil
}

func ValuesFromCli(ctx *cli.Context, inputs Arguments) ([]string, error) {
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

func UnpackAbiData(outTypes Arguments, result []byte) ([]UnpackedValue, error) {
	results := []UnpackedValue{}
	err := UnpackAbiDataWithSetter(outTypes, result, func(i int, r interface{}) {
		results = append(results, UnpackedValue{
			Type:  outTypes[i].Type.String(),
			Value: r,
		})
	})
	return results, err
}

func UnpackAbiDataWithSetter(outTypes Arguments, result []byte, setter func(int, interface{})) error {
	unpackedResults, err := outTypes.Unpack(result)
	if err != nil {
		return err
	} else {
		for i, r := range unpackedResults {
			setter(i, r)
		}
	}
	return nil
}

func TypesFromStrings(typeNames []string) (Arguments, error) {
	var argTypes = make([]Argument, 0, len(typeNames))
	for _, argTypeName := range typeNames {
		if len(argTypeName) == 0 {
			continue
		}
		argType, err := NewType(argTypeName, "", nil) // example: "uint256"
		if err != nil {
			return nil, fmt.Errorf("argument contains invalid type: %s. Error: %w", argTypeName, err)
		}
		arg := Argument{
			Type: argType,
		}
		argTypes = append(argTypes, arg)
	}
	return argTypes, nil
}
