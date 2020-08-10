package main

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/urfave/cli/v2"

	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/sector-storage/database"
)

var hlmStorageCmd = &cli.Command{
	Name:  "hlm-storage",
	Usage: "Manage storage",
	Subcommands: []*cli.Command{
		addHLMStorageCmd,
		disableHLMStorageCmd,
		enableHLMStorageCmd,
		mountHLMStorageCmd,
		umountHLMStorageCmd,
		relinkHLMStorageCmd,
		scaleHLMStorageCmd,
	},
}

var addHLMStorageCmd = &cli.Command{
	Name:  "add",
	Usage: "add a storage node",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "mount-type",
			Usage: "mount type, like nfs, empty for local folder by default.",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "mount-signal-uri",
			Usage: "uri for mount signal channel, net uri or local uri",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "mount-transf-uri",
			Usage: "uri for mount transfer channel, net uri or local uri",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "mount-dir",
			Usage: "parent dir of mount point",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "mount-opt",
			Usage: "mount opt, format should be \"-o ...\"",
			Value: "",
		},
		&cli.Int64Flag{
			Name:  "max-size",
			Usage: "storage max size, in byte",
			Value: 0,
		},
		&cli.Int64Flag{
			Name:  "keep-size",
			Usage: "the storage should keep size for other, in byte",
			Value: 0,
		},
		&cli.Int64Flag{
			Name:  "sector-size",
			Usage: "sector size, the result sizes of sealed+cache, default is 100GB",
			Value: 107374182400,
		},

		&cli.IntFlag{
			Name:  "max-work",
			Usage: "the max number currency work",
			Value: 5,
		},
	},
	Action: func(cctx *cli.Context) error {
		mountType := cctx.String("mount-type")
		mountOpt := cctx.String("mount-opt")
		mountSignalUri := cctx.String("mount-signal-uri")
		if len(mountSignalUri) == 0 {
			return errors.New("need mount-signal-uri")
		}
		mountTransfUri := cctx.String("mount-transf-uri")
		if len(mountTransfUri) == 0 {
			mountTransfUri = mountSignalUri
		}
		mountDir := cctx.String("mount-dir")
		if len(mountDir) == 0 {
			return errors.New("need mount-dir")
		}
		maxSize := cctx.Int64("max-size")
		if maxSize < -1 {
			return errors.New("need max-size")
		}

		keepSize := cctx.Int64("keep-size")
		sectorSize := cctx.Int64("sector-size")
		maxWork := cctx.Int("max-work")
		fmt.Println(mountType, mountSignalUri, mountTransfUri, mountDir, maxSize, keepSize, sectorSize, maxWork)

		nodeApi, closer, err := lcli.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)
		return nodeApi.AddHLMStorage(ctx, database.StorageInfo{
			MountType:      mountType,
			MountSignalUri: mountSignalUri,
			MountTransfUri: mountTransfUri,
			MountDir:       mountDir,
			MountOpt:       mountOpt,
			MaxSize:        maxSize,
			KeepSize:       keepSize,
			SectorSize:     sectorSize,
			MaxWork:        maxWork,
		})
	},
}

var disableHLMStorageCmd = &cli.Command{
	Name:      "disable",
	Usage:     "Disable a storage node to stop allocating",
	ArgsUsage: "id",
	Action: func(cctx *cli.Context) error {
		args := cctx.Args()
		if args.Len() == 0 {
			return errors.New("need input id")
		}
		id, err := strconv.ParseInt(args.First(), 10, 64)
		if err != nil {
			return err
		}
		nodeApi, closer, err := lcli.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)
		return nodeApi.DisableHLMStorage(ctx, id)
	},
}
var enableHLMStorageCmd = &cli.Command{
	Name:      "enable",
	Usage:     "Enable a storage node to recover allocating",
	ArgsUsage: "id",
	Action: func(cctx *cli.Context) error {
		fmt.Println("TODO")
		return nil
	},
}
var mountHLMStorageCmd = &cli.Command{
	Name:      "mount",
	Usage:     "Mount a storage by node id",
	ArgsUsage: "id",
	Action: func(cctx *cli.Context) error {
		args := cctx.Args()
		if args.Len() == 0 {
			return errors.New("need input id")
		}
		id, err := strconv.ParseInt(args.First(), 10, 64)
		if err != nil {
			return err
		}
		nodeApi, closer, err := lcli.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)
		return nodeApi.MountHLMStorage(ctx, id)
	},
}

var umountHLMStorageCmd = &cli.Command{
	Name:      "umount",
	Usage:     "umount a storage by node id or umont all storage",
	ArgsUsage: "[id/all] -- id for one sector, all for all sectors",
	Action: func(cctx *cli.Context) error {
		args := cctx.Args()
		if args.Len() == 0 {
			return errors.New("need input id OR all")
		}
		firstArg := args.First()
		id := int64(0)
		if firstArg != "all" {
			stroageId, err := strconv.ParseInt(firstArg, 10, 64)
			if err != nil {
				return err
			}
			id = stroageId
		}
		nodeApi, closer, err := lcli.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)

		return nodeApi.UMountHLMStorage(ctx, id)
	},
}

var relinkHLMStorageCmd = &cli.Command{
	Name:      "relink",
	Usage:     "Relink the cache and sealed to the storage node",
	ArgsUsage: "[id/all] -- id for one sector, all for all sectors",
	Action: func(cctx *cli.Context) error {
		args := cctx.Args()
		if args.Len() == 0 {
			return errors.New("need input id OR all")
		}
		firstArg := args.First()
		id := int64(0)
		if firstArg != "all" {
			stroageId, err := strconv.ParseInt(firstArg, 10, 64)
			if err != nil {
				return err
			}
			id = stroageId
		}
		nodeApi, closer, err := lcli.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)
		return nodeApi.RelinkHLMStorage(ctx, id)
	},
}

var replaceHLMStorageCmd = &cli.Command{
	Name:  "replace",
	Usage: "Replace the storage node",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:  "storage-id",
			Usage: "id of storage",
		},
		&cli.StringFlag{
			Name:  "mount-signal-uri",
			Usage: "uri for mount signal channel, net uri or local uri who can mount",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "mount-transf-uri",
			Usage: "uri for mount signal channel, net uri or local uri who can mount, empty should same as mount-signal-uri",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "mount-type",
			Usage: "mount type, like nfs, empty to keep the origin value",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "mount-opt",
			Usage: "mount opt, format should be \"-o ...\", empty to keep the origin value",
			Value: "",
		},
	},
	Action: func(cctx *cli.Context) error {
		storageId := cctx.Int64("storage-id")
		if storageId <= 0 {
			return errors.New("need input storage-id>0")
		}
		mountSignalUri := cctx.String("mount-signal-uri")
		if len(mountSignalUri) == 0 {
			return errors.New("need mount-signal-uri")
		}
		mountTransfUri := cctx.String("mount-transf-uri")
		if len(mountTransfUri) == 0 {
			mountTransfUri = mountSignalUri
		}
		mountType := cctx.String("mount-type")
		mountOpt := cctx.String("mount-opt")

		nodeApi, closer, err := lcli.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)
		return nodeApi.ReplaceHLMStorage(ctx, storageId, mountSignalUri, mountTransfUri, mountType, mountOpt)
	},
}

var scaleHLMStorageCmd = &cli.Command{
	Name:  "scale",
	Usage: "scale storage maxSize OR maxWork by node id ",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "storage-id",
			Usage: "storage ID",
			Value: "",
		},
		&cli.Int64Flag{
			Name:  "max-size",
			Usage: "storage max size, in byte",
			Value: 0,
		},
		&cli.IntFlag{
			Name:  "max-work",
			Usage: "the max number currency work",
			Value: 0,
		},
	},
	Action: func(cctx *cli.Context) error {
		storageId, err := strconv.ParseInt(cctx.String("storage-id"), 10, 64)
		if err != nil {
			return err
		}
		if storageId < 1 {
			return errors.New("storageId need input > 1")
		}
		maxSize, err := strconv.ParseInt(cctx.String("max-size"), 10, 64)
		if err != nil {
			return err
		}
		if maxSize < -1 {
			return errors.New("maxSize need input >= -1")
		}
		maxWork, err := strconv.ParseInt(cctx.String("max-work"), 10, 64)
		if err != nil {
			return err
		}
		if maxWork < 0 {
			return errors.New("maxWork need input >= 0")
		}
		nodeApi, closer, err := lcli.GetStorageMinerAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)
		return nodeApi.ScaleHLMStorage(ctx, storageId, maxSize, maxWork)
	},
}
