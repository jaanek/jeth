// Copyright 2018 The go-ethereum Authors
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

package abipack

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ledgerwatch/erigon/accounts/abi"
	"github.com/ledgerwatch/erigon/common"
)

// parseTopicWithSetter converts the indexed topic field-value pairs and stores them using the
// provided set function.
//
// Note, dynamic types cannot be reconstructed since they get mapped to Keccak256
// hashes as the topic value!
func ParseTopicWithSetter(argTypes abi.Arguments, topics []common.Hash, setter func(int, interface{})) error {
	// Sanity check that the fields and topics match up
	if len(argTypes) != len(topics) {
		return errors.New("args/topics count mismatch")
	}
	// Iterate over all the fields and reconstruct them from topics
	for i, arg := range argTypes {
		var reconstr interface{}
		switch arg.Type.T {
		case abi.TupleTy:
			return errors.New("tuple type in topic reconstruction")
		case abi.StringTy, abi.BytesTy, abi.SliceTy, abi.ArrayTy:
			// Array types (including strings and bytes) have their keccak256 hashes stored in the topic- not a hash
			// whose bytes can be decoded to the actual value- so the best we can do is retrieve that hash
			reconstr = topics[i]
		case abi.FunctionTy:
			if garbage := binary.BigEndian.Uint64(topics[i][0:8]); garbage != 0 {
				return fmt.Errorf("bind: got improperly encoded function type, got %v", topics[i].Bytes())
			}
			var tmp [24]byte
			copy(tmp[:], topics[i][8:32])
			reconstr = tmp
		default:
			var err error
			reconstr, err = toGoType(0, arg.Type, topics[i].Bytes())
			if err != nil {
				return err
			}
		}
		// Use the setter function to store the value
		setter(i, reconstr)
	}

	return nil
}
