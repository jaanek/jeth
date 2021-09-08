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
	DataParam = cli.StringFlag{
		Name:  "data",
		Usage: "A hexadecimal data for tx",
	}
	DeployParam = cli.StringFlag{
		Name:  "deploy",
		Usage: "Provide constructor params",
	}
	BinParam = cli.StringFlag{
		Name:  "bin",
		Usage: "Binary data in hex",
	}
	BinFileParam = cli.StringFlag{
		Name:  "bin-file",
		Usage: "Binary data in hex from file",
	}
	MethodParam = cli.StringFlag{
		Name:  "method",
		Usage: "A method call with params",
	}
	Param0 = cli.StringFlag{
		Name:  "0",
		Usage: "",
	}
	Param1 = cli.StringFlag{
		Name:  "1",
		Usage: "",
	}
	Param2 = cli.StringFlag{
		Name:  "2",
		Usage: "",
	}
	Param3 = cli.StringFlag{
		Name:  "3",
		Usage: "",
	}
	Param4 = cli.StringFlag{
		Name:  "4",
		Usage: "",
	}
	Param5 = cli.StringFlag{
		Name:  "5",
		Usage: "",
	}
	Param6 = cli.StringFlag{
		Name:  "6",
		Usage: "",
	}
	Param7 = cli.StringFlag{
		Name:  "7",
		Usage: "",
	}
	Param8 = cli.StringFlag{
		Name:  "8",
		Usage: "",
	}
	Param9 = cli.StringFlag{
		Name:  "9",
		Usage: "",
	}
)
