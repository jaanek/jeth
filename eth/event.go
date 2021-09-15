package eth

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/jaanek/jeth/abi"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/crypto"
)

type EventSpec struct {
	Name      string
	TopicArgs []string
	DataArgs  []string
}

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

type Event interface {
	Name() string
	ParseInto(out interface{}, logs []types.Log) error
}

func NewEvent(eventName string, topicTypes []string, dataTypes []string) (Event, error) {
	topicArgs, err := abi.AbiTypesFromStrings(topicTypes)
	if err != nil {
		return nil, err
	}
	dataArgs, err := abi.AbiTypesFromStrings(dataTypes)
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
		err := abi.ParseTopicWithSetter(e.topicArgs, log.Topics[1:], func(i int, reconstr interface{}) {
			// fmt.Printf("+%v: %v\n", i, reflect.TypeOf(reconstr))
			field := reflect.ValueOf(out).Elem().Field(i)
			field.Set(reflect.ValueOf(reconstr))
		})
		if err != nil {
			return err
		}
		err = abi.UnpackAbiDataWithSetter(e.dataArgs, log.Data, func(i int, reconstr interface{}) {
			// fmt.Printf("-%v: %v\n", len(e.topicArgs)+i, reflect.TypeOf(reconstr))
			field := reflect.ValueOf(out).Elem().Field(len(e.topicArgs) + i)
			field.Set(reflect.ValueOf(reconstr))
		})
		if err != nil {
			return err
		}
	}
	if !match {
		return errors.New("No matching logs")
	}
	return nil
}
