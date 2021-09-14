package abipack

import (
	"fmt"
	"math/big"

	"github.com/holiman/uint256"
	"github.com/ledgerwatch/erigon/accounts/abi"
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

func UnpackAbiData(outTypes abi.Arguments, result []byte) ([]UnpackedValue, error) {
	results := []UnpackedValue{}
	err := UnpackAbiDataWithSetter(outTypes, result, func(i int, r interface{}) {
		results = append(results, UnpackedValue{
			Type:  outTypes[i].Type.String(),
			Value: r,
		})
	})
	return results, err
}

func UnpackAbiDataWithSetter(outTypes abi.Arguments, result []byte, setter func(int, interface{})) error {
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
			Name:    "", // in method calls not used
			Type:    argType,
			Indexed: false, // in method calls not used
		}
		argTypes = append(argTypes, arg)
	}
	return argTypes, nil
}
