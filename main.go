package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jaanek/jeth/commands"
	"github.com/jaanek/jeth/flags"
	"github.com/jaanek/jeth/rpc"
	"github.com/jaanek/jeth/ui"
	"github.com/urfave/cli"
)

type StdInput struct {
	RpcUrl         string `json:"rpcUrl"`
	ChainId        string `json:"chainId"`
	RawTransaction string `json:"tx"`
	TransactionSig string `json:"txsig"`
}

var (
	app = NewApp("eth api command line interface")
)

type Command func(term ui.Screen, ctx *cli.Context) error
type RpcCommand func(term ui.Screen, ctx *cli.Context, endpoint rpc.RpcEndpoint) error

func init() {
	app.Flags = []cli.Flag{}
	app.Commands = []cli.Command{
		{
			Name:    "chain-id",
			Aliases: []string{"chain"},
			Usage:   "returns the chain id of endpoint",
			Action:  rpcCommand(commands.ChainIdCommand),
			Flags: []cli.Flag{
				flags.Verbose,
				flags.RpcUrl,
				flags.Gwei,
			},
		},
		{
			Name:    "block-number",
			Aliases: []string{"bn"},
			Usage:   "returns the number of most recent block",
			Action:  rpcCommand(commands.BlockNumberCommand),
			Flags: []cli.Flag{
				flags.Verbose,
				flags.RpcUrl,
			},
		},
		{
			Name:    "gas-price",
			Aliases: []string{"gp"},
			Usage:   "returns the current price per gas in wei",
			Action:  rpcCommand(commands.GasPriceCommand),
			Flags: []cli.Flag{
				flags.Verbose,
				flags.RpcUrl,
				flags.Gwei,
			},
		},
		{
			Name:   "tip",
			Usage:  "returns a suggestion for a gas tip cap for dynamic fee transactions",
			Action: rpcCommand(commands.MaxPriorityFeePerGasCommand),
			Flags: []cli.Flag{
				flags.Verbose,
				flags.RpcUrl,
				flags.Gwei,
			},
		},
		{
			Name:    "tx-params",
			Aliases: []string{"params"},
			Usage:   "returns transaction params, nonce, prices, gas, etc required for signing a tx",
			Action:  rpcCommand(commands.TransactionParamsCommand),
			Flags: []cli.Flag{
				flags.Verbose,
				flags.Plain,
				flags.RpcUrl,
				flags.FromParam,
				flags.ToParam,
				flags.ValueParam,
				flags.ValueInEthParam,
				flags.ValueInGweiParam,
				flags.DataParam,
				flags.DeployParam,
				flags.BinParam,
				flags.BinFileParam,
				flags.MethodParam,
				flags.Param0,
				flags.Param1,
				flags.Param2,
				flags.Param3,
				flags.Param4,
				flags.Param5,
				flags.Param6,
				flags.Param7,
				flags.Param8,
				flags.Param9,
			},
		},
		{
			Name:   "balance",
			Usage:  "get account balance",
			Action: rpcCommand(commands.GetAccountBalanceCommand),
			Flags: []cli.Flag{
				flags.Verbose,
				flags.RpcUrl,
				flags.HexParam,
			},
		},
		{
			Name:    "estimate-gas",
			Aliases: []string{"estimate"},
			Usage:   "get estimated gas used by a tx",
			Action:  rpcCommand(commands.EstimateGasCommand),
			Flags: []cli.Flag{
				flags.Verbose,
				flags.RpcUrl,
				flags.FromParam,
				flags.ToParam,
				flags.ValueParam,
				flags.ValueInEthParam,
				flags.ValueInGweiParam,
				flags.DataParam,
			},
		},
		{
			Name:    "tx-count",
			Aliases: []string{"count"},
			Usage:   "get transactions count for the from address",
			Action:  rpcCommand(commands.TransactionsCountCommand),
			Flags: []cli.Flag{
				flags.Verbose,
				flags.RpcUrl,
				flags.HexParam,
			},
		},
		{
			Name:    "tx-send",
			Aliases: []string{"send"},
			Usage:   "sends previously signed transaction (message call or contract creation) to endpoint. Returns tx hash",
			Action:  rpcCommand(commands.SendTransactionCommand),
			Flags: []cli.Flag{
				flags.Verbose,
				flags.RpcUrl,
				flags.TxParam,
			},
		},
		{
			Name:   "receipt",
			Usage:  "get transaction receipt",
			Action: rpcCommand(commands.GetTransactionReceiptCommand),
			Flags: []cli.Flag{
				flags.Verbose,
				flags.RpcUrl,
				flags.HexParam,
			},
		},
		{
			Name:   "pack-values",
			Usage:  "packs method values",
			Action: runCommand(commands.PackValuesCommand),
			Flags: []cli.Flag{
				flags.Verbose,
				flags.Plain,
				flags.MethodParam,
				flags.Param0,
				flags.Param1,
				flags.Param2,
				flags.Param3,
				flags.Param4,
				flags.Param5,
				flags.Param6,
				flags.Param7,
				flags.Param8,
				flags.Param9,
			},
		},
	}
}

func runCommand(cmd Command) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		term := ui.NewTerminal(ctx.Bool(flags.Verbose.Name))
		err := cmd(term, ctx)
		if err != nil {
			term.Error(err)
		}
		return nil
	}
}

func rpcCommand(cmd RpcCommand) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		term := ui.NewTerminal(ctx.Bool(flags.Verbose.Name))
		var rpcUrl string
		if ctx.IsSet(flags.RpcUrl.Name) {
			rpcUrl = ctx.String(flags.RpcUrl.Name)
		} else if flags.FlagRpcUrl != nil && *flags.FlagRpcUrl != "" {
			rpcUrl = *flags.FlagRpcUrl
		} else {
			return errors.New(fmt.Sprintf("Missing --%s", flags.RpcUrl.Name))
		}
		endpoint := rpc.NewEndpoint(rpcUrl)
		err := cmd(term, ctx, endpoint)
		if err != nil {
			term.Error(err)
		}
		return nil
	}
}

func main() {
	// try to read command params from from std input json stream
	if isReadFromStdInArgSpecified(os.Args) {
		stdInStr := StdInReadAll()
		if len(stdInStr) > 0 {
			input := StdInput{}
			err := json.Unmarshal([]byte(stdInStr), &input)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while parsing stdin json: %s\n", err)
			} else {
				flags.FlagRpcUrl = &input.RpcUrl
				flags.FlagRawTx = &input.RawTransaction
			}
		}
	}
	if err := app.Run(os.Args); err != nil {
		code := 1
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}

// NewApp creates an app with sane defaults.
func NewApp(usage string) *cli.App {
	app := cli.NewApp()
	app.Name = filepath.Base(os.Args[0])
	app.Author = ""
	app.Email = ""
	app.Usage = usage
	return app
}

func isReadFromStdInArgSpecified(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			return true
		}
	}
	return false
}

func StdInReadAll() string {
	arr := make([]string, 0)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		scanner.Scan()
		text := scanner.Text()
		if len(text) > 0 {
			arr = append(arr, text)
		} else {
			break
		}
	}
	return strings.Join(arr, "")
}
