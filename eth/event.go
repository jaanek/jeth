package eth

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/jaanek/jeth/abipack"
	"github.com/ledgerwatch/erigon/accounts/abi"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/crypto"
)

type HashedEvent struct {
	Sig string
	Id  []byte
}

// https://docs.soliditylang.org/en/develop/abi-spec.html
func NewHashedEvent(eventName string, topicArgs abi.Arguments, dataArgs abi.Arguments) HashedEvent {
	var types = make([]string, len(topicArgs)+len(dataArgs))
	var i = 0
	for _, input := range topicArgs {
		types[i] = input.Type.String()
		i++
	}
	for _, input := range dataArgs {
		types[i] = input.Type.String()
		i++
	}
	event := HashedEvent{}
	event.Sig = fmt.Sprintf("%v(%v)", eventName, strings.Join(types, ","))
	sig := crypto.Keccak256([]byte(event.Sig))
	event.Id = make([]byte, len(sig))
	copy(event.Id, sig)
	return event
}

type event struct {
	eventName string
	hash      HashedEvent
	topicArgs abi.Arguments
	dataArgs  abi.Arguments
}

func (m *event) Name() string {
	return m.eventName
}

func (m *event) TopicArgs() abi.Arguments {
	return m.topicArgs
}

func (m *event) DataArgs() abi.Arguments {
	return m.dataArgs
}

type Event interface {
	Name() string
	TopicArgs() abi.Arguments
	DataArgs() abi.Arguments
	ParseInto(out interface{}, logs []types.Log) error
}

func NewEvent(eventName string, topicTypes []string, dataTypes []string) (Event, error) {
	topicArgs, err := abipack.AbiTypesFromStrings(topicTypes)
	if err != nil {
		return nil, err
	}
	dataArgs, err := abipack.AbiTypesFromStrings(dataTypes)
	if err != nil {
		return nil, err
	}
	hashed := NewHashedEvent(eventName, topicArgs, dataArgs)
	return &event{
		eventName: eventName,
		hash:      hashed,
		topicArgs: topicArgs,
		dataArgs:  dataArgs,
	}, nil
}

func (e *event) ParseInto(out interface{}, logs []types.Log) error {
	var match bool
	for _, log := range logs {
		eventHash := log.Topics[0]
		if bytes.Compare(eventHash.Bytes(), e.hash.Id) != 0 {
			continue
		}
		match = true
		err := abipack.ParseTopicWithSetter(e.topicArgs, log.Topics[1:], func(i int, reconstr interface{}) {
			fmt.Printf("+%v: %v", i, reflect.TypeOf(reconstr))
			field := reflect.ValueOf(out).Elem().Field(i)
			field.Set(reflect.ValueOf(reconstr))
		})
		if err != nil {
			return err
		}
		err = abipack.UnpackAbiDataWithSetter(e.dataArgs, log.Data, func(i int, reconstr interface{}) {
			fmt.Printf("-%v: %v", len(e.topicArgs)+i, reflect.TypeOf(reconstr))
			field := reflect.ValueOf(out).Elem().Field(len(e.topicArgs) + i)
			field.Set(reflect.ValueOf(reconstr))
		})
		if err != nil {
			return err
		}
		// if err := args.Copy(event, unpacked); err != nil {
		// 	return err
		// }
		// term.Print(fmt.Sprintf("Event: %+v, log: %+v, data: %v\n", event, log, unpacked))
	}
	if !match {
		return errors.New("No matching logs")
	}
	return nil
}
