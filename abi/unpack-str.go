// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package abi

import (
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	"github.com/ledgerwatch/erigon/common"
	"github.com/ledgerwatch/erigon/common/hexutil"
)

// toGoType parses the output bytes and recursively assigns the value of these bytes
// into a go type with accordance with the ABI spec.
func ToGoTypeFromStr(t Type, input string) (interface{}, error) {
	switch t.T {
	case TupleTy:
		if isDynamicType(t) {
			// begin, err := tuplePointsTo(index, output)
			// if err != nil {
			// 	return nil, err
			// }
			// return forTupleUnpack(t, output[begin:])
			return nil, fmt.Errorf("abi: unimplemented: %v", input)
		}
		return forTupleUnpackFromStr(t, input)
	case SliceTy, ArrayTy:
		return forEachUnpackFromStr(t, input)
	case StringTy:
		return input, nil
	case IntTy, UintTy:
		return ReadIntegerFromStr(t, input)
	case BoolTy:
		return readBoolFromStr(input)
	case AddressTy:
		b, err := hexutil.Decode(input)
		if err != nil {
			return nil, err
		}
		return common.BytesToAddress(b), nil
	case HashTy:
		b, err := hexutil.Decode(input)
		if err != nil {
			return nil, err
		}
		return common.BytesToHash(b), nil
	case BytesTy:
		return hexutil.Decode(input)
	case FixedBytesTy:
		return ReadFixedBytesFromStr(t, input)
	// case abi.FunctionTy:
	// 	return readFunctionType(t, returnOutput)
	default:
		return nil, fmt.Errorf("abi: unknown type %v", t.T)
	}
}

func ReadIntegerFromStr(typ Type, input string) (interface{}, error) {
	// num, ok := math.ParseBig256(input)
	num, ok := new(big.Int).SetString(input, 10)
	if !ok {
		return nil, fmt.Errorf("abi: cannot parse provided integer to big.Int, provided: %v", input)
	}
	// validate input length
	switch typ.Size {
	case 8:
		if num.BitLen() > 8 {
			return nil, fmt.Errorf("abi: input not a byte (8bit): %v", input)
		}
	case 16:
		if num.BitLen() > 16 {
			return nil, fmt.Errorf("abi: input not a word (16bit): %v", input)
		}
	case 32:
		if num.BitLen() > 32 {
			return nil, fmt.Errorf("abi: input not a 32bit integer: %v", input)
		}
	case 64:
		if num.BitLen() > 64 {
			return nil, fmt.Errorf("abi: input not a 64bit integer: %v", input)
		}
	case 256:
		if num.BitLen() > 256 {
			return nil, fmt.Errorf("abi: input not a 64bit integer: %v", input)
		}
	default:
		return nil, fmt.Errorf("abi: unknown bit length of integer. Type bit length:%v, provided input: %v", typ.Size, input)
	}
	if typ.T == UintTy {
		switch typ.Size {
		case 8:
			return uint8(num.Int64()), nil
		case 16:
			return uint16(num.Int64()), nil
		case 32:
			return uint32(num.Int64()), nil
		case 64:
			return uint64(num.Int64()), nil
		case 256:
			return num, nil
		}
		switch typ.Size {
		case 8:
			return int8(num.Int64()), nil
		case 16:
			return int16(num.Int64()), nil
		case 32:
			return int32(num.Int64()), nil
		case 64:
			return int64(num.Int64()), nil
		case 256:
			return num, nil
		}
	}
	return nil, fmt.Errorf("abi: unimplemented, provided input: %v", input)
}

func readBoolFromStr(input string) (bool, error) {
	return strconv.ParseBool(input)
}

// ReadFixedBytes uses reflection to create a fixed array to be read from.
func ReadFixedBytesFromStr(t Type, input string) (interface{}, error) {
	if t.T != FixedBytesTy {
		return nil, fmt.Errorf("abi: invalid type in call to make fixed byte array")
	}
	b, err := hexutil.Decode(input)
	if err != nil {
		return nil, err
	}
	array := reflect.New(t.GetType()).Elem()
	reflect.Copy(array, reflect.ValueOf(b))
	return array.Interface(), nil

}

// forEachUnpack iteratively unpack elements.
func forEachUnpackFromStr(t Type, input string) (interface{}, error) {
	args := strings.Split(input, ",")
	if len(args) == 0 {
		return nil, fmt.Errorf("abi: no array of input args specified. Example: item,item,...")
	}

	// this value will become our slice or our array, depending on the type
	var refSlice reflect.Value

	if t.T == SliceTy {
		// declare our slice
		refSlice = reflect.MakeSlice(t.GetType(), len(args), len(args))
	} else if t.T == ArrayTy {
		// declare our array
		refSlice = reflect.New(t.GetType()).Elem()
	} else {
		return nil, fmt.Errorf("abi: invalid type in array/slice unpacking stage")
	}

	// Arrays have packed elements, resulting in longer unpack steps.
	// Slices have just 32 bytes per element (pointing to the contents).
	// elemSize := getTypeSize(*t.Elem)
	for i, arg := range args {
		inter, err := ToGoTypeFromStr(*t.Elem, arg)
		if err != nil {
			return nil, err
		}
		// append the item to our reflect slice
		refSlice.Index(i).Set(reflect.ValueOf(inter))
	}

	// return the interface
	return refSlice.Interface(), nil
}

func forTupleUnpackFromStr(t Type, input string) (interface{}, error) {
	retval := reflect.New(t.GetType()).Elem()
	virtualArgs := 0
	args := strings.Split(input, ",")
	if len(args) == 0 {
		return nil, fmt.Errorf("abi: no array of input args specified. Example: item,item,...")
	}
	if len(args) != len(t.TupleElems) {
		return nil, fmt.Errorf("abi: provided args != t.TupleElems")
	}
	for index, elem := range t.TupleElems {
		marshalledValue, err := ToGoTypeFromStr(*elem, args[index])
		if elem.T == ArrayTy && !isDynamicType(*elem) {
			// If we have a static array, like [3]uint256, these are coded as
			// just like uint256,uint256,uint256.
			// This means that we need to add two 'virtual' arguments when
			// we count the index from now on.
			//
			// Array values nested multiple levels deep are also encoded inline:
			// [2][3]uint256: uint256,uint256,uint256,uint256,uint256,uint256
			//
			// Calculate the full array size to get the correct offset for the next argument.
			// Decrement it by 1, as the normal index increment is still applied.
			virtualArgs += getTypeSize(*elem)/32 - 1
		} else if elem.T == TupleTy && !isDynamicType(*elem) {
			// If we have a static tuple, like (uint256, bool, uint256), these are
			// coded as just like uint256,bool,uint256
			virtualArgs += getTypeSize(*elem)/32 - 1
		}
		if err != nil {
			return nil, err
		}
		retval.Field(index).Set(reflect.ValueOf(marshalledValue))
	}
	return retval.Interface(), nil
}
