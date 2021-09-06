package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jaanek/jeth/cmd/commands"
	"github.com/jaanek/jeth/flags"
	"github.com/jaanek/jeth/ui"
	"github.com/urfave/cli"
)

var (
	app = NewApp("eth api command line interface")
)

func init() {
	app.Flags = []cli.Flag{}
	app.Commands = []cli.Command{
		{
			Name:    "block-number",
			Aliases: []string{"bn"},
			Usage:   "returns the number of most recent block",
			Action:  runCommand(commands.BlockNumberCommand),
			Flags: []cli.Flag{
				flags.RpcUrlFlag,
			},
		},
	}
}

func runCommand(cmd func(term ui.Screen, ctx *cli.Context) error) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		term := ui.NewTerminal(ctx.Bool(flags.FlagQuiet.Name))
		err := cmd(term, ctx)
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
