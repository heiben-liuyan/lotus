package main

import (
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/gwaylib/errors"
	"github.com/urfave/cli/v2"
)

var testingCmd = &cli.Command{
	Name:  "testing",
	Usage: "run a method testing, see: storage/testing.go",
	Action: func(cctx *cli.Context) error {
		nodeApi, closer, err := lcli.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)
		args := cctx.Args().Slice()
		if len(args) < 1 {
			return errors.New("need function name.")
		}

		return nodeApi.Testing(ctx, args[0], args[1:])
	},
}
