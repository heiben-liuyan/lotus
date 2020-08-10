package main

import (
	_ "net/http/pprof"
	"os"
	"strings"

	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/lotus/build"
	lcli "github.com/filecoin-project/lotus/cli"
)

var log = logging.Logger("chainwatch")

func main() {
	logging.SetLogLevel("*", "INFO")

	log.Info("Starting chainwatch")

	local := []*cli.Command{
		runCmd,
	}

	app := &cli.App{
		Name:    "lotus-chainwatch",
		Usage:   "Devnet token distribution utility",
		Version: build.UserVersion(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "repo",
				EnvVars: []string{"LOTUS_PATH"},
				Value:   "~/.lotus", // TODO: Consider XDG_DATA_HOME
			},
		},

		Commands: local,
	}

	if err := app.Run(os.Args); err != nil {
		log.Warnf("%+v", err)
		return
	}
}

var runCmd = &cli.Command{
	Name:  "run",
	Usage: "Start lotus chainwatch",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "max-batch",
			Value: 1000,
		},
		&cli.StringFlag{
			Name:  "kafka-cert",
			Value: "",
			Usage: "File path for kafka cert",
		},
		&cli.StringFlag{
			Name:  "kafka-addr",
			Value: "",
			Usage: "kafka server address, seperate by space, using like: --kafka-addr=\"addr1 addr2 add3\"",
		},
		&cli.StringFlag{
			Name:  "kafka-topic",
			Value: "",
			Usage: "kafka server topic",
		},

		&cli.StringFlag{
			Name:  "kafka-user",
			Value: "",
			Usage: "username for kafka server",
		},
		&cli.StringFlag{
			Name:  "kafka-pwd",
			Value: "",
			Usage: "password for kafka server",
		},
	},
	Action: func(cctx *cli.Context) error {
		api, closer, err := lcli.GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer closer()
		ctx := lcli.ReqContext(cctx)

		v, err := api.Version(ctx)
		if err != nil {
			return err
		}

		log.Infof("Remote version: %s", v.Version)

		_kafkaCertFile = cctx.String("kafka-cert")
		_kafkaUser = cctx.String("kafka-user")
		_kafkaPasswd = cctx.String("kafka-pwd")
		_kafkaAddress = strings.Split(cctx.String("kafka-addr"), " ")
		_kafkaTopic = cctx.String("kafka-topic")

		InitDB(cctx.String("repo"))
		runSyncer(ctx, api)

		<-ctx.Done()
		os.Exit(0)
		return nil
	},
}
