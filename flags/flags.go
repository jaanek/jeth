package flags

import "github.com/urfave/cli"

var (
	FlagRpcUrl *string
	FlagRawTx  *string
)

var (
	RpcUrl = cli.StringFlag{
		Name:        "rpc.url",
		Usage:       "Rpc endpoint url",
		Destination: FlagRpcUrl,
	}
	Verbose = cli.BoolFlag{
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
	Plain = cli.BoolFlag{
		Name:  "plain",
		Usage: "output as plain text",
	}
	HexParam = cli.StringFlag{
		Name:  "param",
		Usage: "provide rpc param in hex format (starts with 0x)",
	}
	TxParam = cli.StringFlag{
		Name:        "tx",
		Usage:       "provide a raw tx in hex format (starts with 0x)",
		Destination: FlagRawTx,
	}
	FromParam = cli.StringFlag{
		Name:  "from",
		Usage: "provide from address in hex format (starts with 0x)",
	}
	ToParam = cli.StringFlag{
		Name:  "to",
		Usage: "provide to address in hex format (starts with 0x)",
	}
	ValueParam = cli.StringFlag{
		Name:  "value",
		Usage: "in wei",
	}
	ValueInEthParam = cli.BoolFlag{
		Name:  "value-eth",
		Usage: "indicate that provided --value is in eth and not in wei",
	}
	ValueInGweiParam = cli.BoolFlag{
		Name:  "value-gwei",
		Usage: "indicate that provided --value is in gwei and not in wei",
	}
	InputParam = cli.StringFlag{
		Name:  "input",
		Usage: "A hexadecimal input data for tx",
	}
)
