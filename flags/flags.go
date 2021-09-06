package flags

import "github.com/urfave/cli"

var (
	RpcUrlFlag = cli.StringFlag{
		Name:  "rpc.url",
		Usage: "Rpc endpoint url",
	}
	FlagQuiet = cli.BoolFlag{
		Name:  "quiet",
		Usage: "be quiet when outputting results",
	}
)
