package commands

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jaanek/jeth/flags"
	"github.com/jaanek/jeth/ui"
	"github.com/urfave/cli"
)

type PackedValuesOutput struct {
	MethodSig    string `json:"methodSig"`
	MethodHashed string `json:"methodHashed"`
	PackedValues string `json:packedValues`
}

func PackValuesCommand(term ui.Screen, ctx *cli.Context) error {
	if !ctx.IsSet(flags.MethodParam.Name) {
		return errors.New(fmt.Sprintf("Missing method param --%s", flags.MethodParam.Name))
	}
	errMsg := fmt.Sprintf("Method call needs to be specified in format (example): --%s=transfer:address,uint256", flags.MethodParam.Name)
	methodStr := ctx.String(flags.MethodParam.Name)
	methodSplit := strings.Split(methodStr, ":")
	if len(methodSplit) != 2 {
		return errors.New(errMsg)
	}
	methodName := methodSplit[0]
	typeNames := strings.Split(methodSplit[1], ",")
	if len(typeNames) == 0 {
		return errors.New(errMsg)
	}
	argTypes, packedValues, err := abiPackedValuesFromCli(ctx, typeNames)
	if err != nil {
		return err
	}
	method := NewHashedMethod(methodName, argTypes)

	// output
	hashed := make([]byte, len(method.Id)*2)
	packed := make([]byte, len(packedValues)*2)
	hex.Encode(hashed, method.Id[:])
	hex.Encode(packed, packedValues)
	out := PackedValuesOutput{
		MethodSig:    method.Sig,
		MethodHashed: string(hashed),
		PackedValues: string(packed),
	}
	if ctx.IsSet(flags.Plain.Name) {
		term.Print(fmt.Sprintf("method signature: %s", out.MethodSig))
		term.Print(fmt.Sprintf("hashed method: %s", out.MethodHashed))
		term.Print(fmt.Sprintf("packed values: %s", out.PackedValues))
	}
	b, err := json.Marshal(&out)
	if err != nil {
		return err
	}
	term.Output(fmt.Sprintf("%s\n", string(b)))
	return nil
}
