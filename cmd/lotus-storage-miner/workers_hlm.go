package main

import (
	"fmt"
	"strconv"

	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/gwaylib/errors"
	"github.com/urfave/cli/v2"
)

var hlmWorkerCmd = &cli.Command{
	Name:  "hlm-worker",
	Usage: "Manage worker",
	Subcommands: []*cli.Command{
		infoHLMWorkerCmd,
		listHLMWorkerCmd,
		disableHLMWorkerCmd,
	},
}
var disableHLMWorkerCmd = &cli.Command{
	Name:      "disable",
	Usage:     "Disable a work node to stop allocating OR start allocating",
	ArgsUsage: "id and disable value",
	Action: func(cctx *cli.Context) error {
		args := cctx.Args()
		if args.Len() < 2 {
			return errors.New("need input workid AND disable value")
		}
		workerId := args.First()
		disable, err := strconv.ParseBool(args.Get(1))
		if err != nil {
			return errors.New("need input disable true/false")
		}
		nodeApi, closer, err := lcli.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)
		return nodeApi.WorkerDisable(ctx, workerId, disable)
	},
}

var infoHLMWorkerCmd = &cli.Command{
	Name:  "info",
	Usage: "worker information",
	Action: func(cctx *cli.Context) error {
		nodeApi, closer, err := lcli.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)
		wstat, err := nodeApi.WorkerStatus(ctx)
		if err != nil {
			return err
		}

		fmt.Printf("Worker use:\n")
		fmt.Printf("\tLocal: %d / %d (+%d reserved)\n", wstat.LocalTotal-wstat.LocalReserved-wstat.LocalFree, wstat.LocalTotal-wstat.LocalReserved, wstat.LocalReserved)
		fmt.Printf("\tSealWorker: %d / %d (locked: %d)\n", wstat.SealWorkerUsing, wstat.SealWorkerTotal, wstat.SealWorkerLocked)
		fmt.Printf("\tCommit2Srv: %d / %d\n", wstat.Commit2SrvUsed, wstat.Commit2SrvTotal)
		fmt.Printf("\tWnPoStSrv : %d / %d\n", wstat.WnPoStSrvUsed, wstat.WnPoStSrvTotal)
		fmt.Printf("\tWdPoStSrv : %d / %d\n", wstat.WdPoStSrvUsed, wstat.WdPoStSrvTotal)

		fmt.Printf("Queues:\n")
		fmt.Printf("\tAddPiece: %d\n", wstat.AddPieceWait)
		fmt.Printf("\tPreCommit1: %d\n", wstat.PreCommit1Wait)
		fmt.Printf("\tPreCommit2: %d\n", wstat.PreCommit2Wait)
		fmt.Printf("\tCommit1: %d\n", wstat.Commit1Wait)
		fmt.Printf("\tCommit2: %d\n", wstat.Commit2Wait)
		fmt.Printf("\tFinalize: %d\n", wstat.FinalizeWait)
		fmt.Printf("\tUnseal: %d\n", wstat.UnsealWait)
		return nil
	},
}
var listHLMWorkerCmd = &cli.Command{
	Name:  "list",
	Usage: "list worker status",
	Action: func(cctx *cli.Context) error {
		nodeApi, closer, err := lcli.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)
		infos, err := nodeApi.WorkerStatusAll(ctx)
		if err != nil {
			return errors.As(err)
		}
		for _, info := range infos {
			fmt.Println(info.String())
		}
		return nil
	},
}
