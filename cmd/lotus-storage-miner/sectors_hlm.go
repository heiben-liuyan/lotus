package main

import (
	"fmt"

	"github.com/urfave/cli/v2"

	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/gwaylib/errors"
)

var pledgeSectorCmd = &cli.Command{
	Name:  "pledge-sector",
	Usage: "Pledge sector",
	Subcommands: []*cli.Command{
		startPledgeSectorCmd,
		statusPledgeSectorCmd,
		stopPledgeSectorCmd,
	},
}

var hlmSectorCmd = &cli.Command{
	Name:  "hlm-sector",
	Usage: "command for hlm-sector",
	Subcommands: []*cli.Command{
		setHlmSectorStateCmd,
	},
}

var startPledgeSectorCmd = &cli.Command{
	Name:  "start",
	Usage: "start the pledge daemon",
	Action: func(cctx *cli.Context) error {
		nodeApi, closer, err := lcli.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)

		return nodeApi.RunPledgeSector(ctx)
	},
}

var statusPledgeSectorCmd = &cli.Command{
	Name:  "status",
	Usage: "the pledge daemon status",
	Action: func(cctx *cli.Context) error {
		nodeApi, closer, err := lcli.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)

		if status, err := nodeApi.StatusPledgeSector(ctx); err != nil {
			return errors.As(err)
		} else if status != 0 {
			fmt.Println("Running")
		} else {
			fmt.Println("Not Running")
		}
		return nil
	},
}
var stopPledgeSectorCmd = &cli.Command{
	Name:  "stop",
	Usage: "stop the pledge daemon",
	Action: func(cctx *cli.Context) error {
		nodeApi, closer, err := lcli.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)

		return nodeApi.StopPledgeSector(ctx)
	},
}

var setHlmSectorStateCmd = &cli.Command{
	Name:  "set-state",
	Usage: "will set the sector state",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "sector-id",
			Usage: "sector id which want to set",
		},
		&cli.IntFlag{
			Name:  "state",
			Usage: "state which want to set",
		},
		&cli.StringFlag{
			Name:  "memo",
			Usage: "memo for state udpate",
		},
	},
	Action: func(cctx *cli.Context) error {
		nodeApi, closer, err := lcli.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)

		sid := cctx.String("sector-id")
		if len(sid) == 0 {
			return errors.New("need input sector-id")
		}
		memo := cctx.String("memo")
		if len(memo) == 0 {
			return errors.New("need input memo")
		}
		return nodeApi.HlmSectorSetState(ctx, sid, memo, cctx.Int("state"))
	},
}
var finalizeHlmSectorCmd = &cli.Command{
	Name:  "finalize",
	Usage: "will call the finalize to trigger the remote do finalize",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "sector-id",
			Usage: "sector id which want to set",
		},
	},
	Action: func(cctx *cli.Context) error {
		nodeApi, closer, err := lcli.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)

		sid := cctx.String("sector-id")
		if len(sid) == 0 {
			return errors.New("need input sector-id")
		}
		return nodeApi.HlmSectorFinalize(ctx, sid)
	},
}
