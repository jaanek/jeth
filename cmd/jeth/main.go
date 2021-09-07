package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jaanek/jeth/cmd/commands"
	"github.com/jaanek/jeth/flags"
	"github.com/jaanek/jeth/rpc"
	"github.com/jaanek/jeth/ui"
	"github.com/urfave/cli"
)

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
				flags.FlagVerbose,
				flags.RpcUrlFlag,
				flags.Gwei,
			},
		},
		{
			Name:    "block-number",
			Aliases: []string{"bn"},
			Usage:   "returns the number of most recent block",
			Action:  rpcCommand(commands.BlockNumberCommand),
			Flags: []cli.Flag{
				flags.FlagVerbose,
				flags.RpcUrlFlag,
			},
		},
		{
			Name:    "gas-price",
			Aliases: []string{"gp"},
			Usage:   "returns the current price per gas in wei",
			Action:  rpcCommand(commands.GasPriceCommand),
			Flags: []cli.Flag{
				flags.FlagVerbose,
				flags.RpcUrlFlag,
				flags.Gwei,
			},
		},
		{
			Name:   "tip",
			Usage:  "returns a suggestion for a gas tip cap for dynamic fee transactions",
			Action: rpcCommand(commands.MaxPriorityFeePerGasCommand),
			Flags: []cli.Flag{
				flags.FlagVerbose,
				flags.RpcUrlFlag,
				flags.Gwei,
			},
		},
		{
			Name:    "tx-params",
			Aliases: []string{"params"},
			Usage:   "returns transaction params, nonce, prices, etc",
			Action:  rpcCommand(commands.TransactionParamsCommand),
			Flags: []cli.Flag{
				flags.FlagVerbose,
				flags.RpcUrlFlag,
				flags.Gwei,
				flags.Eth,
				flags.Plain,
				flags.FromParam,
			},
		},
		{
			Name:   "balance",
			Usage:  "get account balance",
			Action: rpcCommand(commands.GetAccountBalanceCommand),
			Flags: []cli.Flag{
				flags.FlagVerbose,
				flags.RpcUrlFlag,
				flags.HexParam,
			},
		},
		{
			Name:    "estimate-gas",
			Aliases: []string{"estimate"},
			Usage:   "get estimated gas used by a tx",
			Action:  rpcCommand(commands.EstimateGasCommand),
			Flags: []cli.Flag{
				flags.FlagVerbose,
				flags.RpcUrlFlag,
				flags.HexParam,
			},
		},
		{
			Name:    "tx-count",
			Aliases: []string{"count"},
			Usage:   "get transactions count for the from address",
			Action:  rpcCommand(commands.TransactionsCountCommand),
			Flags: []cli.Flag{
				flags.FlagVerbose,
				flags.RpcUrlFlag,
				flags.HexParam,
			},
		},
		{
			Name:    "tx-send",
			Aliases: []string{"send"},
			Usage:   "sends previously signed transaction (message call or contract creation) to endpoint. Returns tx hash",
			Action:  rpcCommand(commands.SendTransactionCommand),
			Flags: []cli.Flag{
				flags.FlagVerbose,
				flags.RpcUrlFlag,
				flags.HexParam,
				flags.StdIn,
			},
		},
		{
			Name:   "receipt",
			Usage:  "get transaction receipt",
			Action: rpcCommand(commands.GetTransactionReceiptCommand),
			Flags: []cli.Flag{
				flags.FlagVerbose,
				flags.RpcUrlFlag,
				flags.HexParam,
			},
		},
	}
}

func runCommand(cmd Command) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		term := ui.NewTerminal(ctx.Bool(flags.FlagVerbose.Name))
		err := cmd(term, ctx)
		if err != nil {
			term.Error(err)
		}
		return nil
	}
}

func rpcCommand(cmd RpcCommand) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		term := ui.NewTerminal(ctx.Bool(flags.FlagVerbose.Name))
		if !ctx.IsSet(flags.RpcUrlFlag.Name) {
			return errors.New(fmt.Sprintf("Missing --%s", flags.RpcUrlFlag.Name))
		}
		endpoint := rpc.NewEndpoint(ctx.String(flags.RpcUrlFlag.Name))
		err := cmd(term, ctx, endpoint)
		if err != nil {
			term.Error(err)
		}
		return nil
	}
}

func main() {
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
