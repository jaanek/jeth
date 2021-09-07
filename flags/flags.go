package flags

import "github.com/urfave/cli"

var (
	RpcUrlFlag = cli.StringFlag{
		Name:  "rpc.url",
		Usage: "Rpc endpoint url",
	}
	FlagVerbose = cli.BoolFlag{
		Name:  "verbose",
		Usage: "output debug information",
	}
	StdIn = cli.BoolFlag{
		Name:  "stdin",
		Usage: "read input from standard input",
	}
	Gwei = cli.BoolFlag{
		Name:  "gwei",
		Usage: "output in gwei's",
	}
	Eth = cli.BoolFlag{
		Name:  "eth",
		Usage: "output in eth's",
	}
	Plain = cli.BoolFlag{
		Name:  "plain",
		Usage: "output as plain text",
	}
	HexParam = cli.StringFlag{
		Name:  "param",
		Usage: "provide rpc param in hex format (starts with 0x)",
	}
	FromParam = cli.StringFlag{
		Name:  "from",
		Usage: "provide rpc param in hex format (starts with 0x)",
	}
)
