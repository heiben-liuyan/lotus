package main

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	mux "github.com/gorilla/mux"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-jsonrpc/auth"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/apistruct"
	"github.com/filecoin-project/lotus/build"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/lotus/lib/report"
	"github.com/filecoin-project/lotus/lib/ulimit"
	"github.com/filecoin-project/lotus/node"
	"github.com/filecoin-project/lotus/node/impl"
	"github.com/filecoin-project/lotus/node/modules/dtypes"
	"github.com/filecoin-project/lotus/node/repo"

	"github.com/filecoin-project/lotus/lib/fileserver"
	"github.com/filecoin-project/sector-storage/database"
	"github.com/filecoin-project/sector-storage/ffiwrapper"
	"github.com/gwaylib/errors"
)

var runCmd = &cli.Command{
	Name:  "run",
	Usage: "Start a lotus miner process",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "api",
			Usage: "2345",
		},
		&cli.StringFlag{
			Name:  "report-url",
			Value: "",
			Usage: "report url for state",
		},
		&cli.BoolFlag{
			Name:  "enable-gpu-proving",
			Usage: "enable use of GPU for mining operations",
			Value: true,
		},
		&cli.BoolFlag{
			Name:  "nosync",
			Usage: "don't check full-node sync status",
		},
		&cli.BoolFlag{
			Name:  "manage-fdlimit",
			Usage: "manage open file limit",
			Value: true,
		},
	},
	Action: func(cctx *cli.Context) error {
		if !cctx.Bool("enable-gpu-proving") {
			os.Setenv("BELLMAN_NO_GPU", "true")
		}

		nodeApi, ncloser, err := lcli.GetFullNodeAPI(cctx)
		if err != nil {
			return err
		}
		defer ncloser()
		ctx := lcli.DaemonContext(cctx)

		v, err := nodeApi.Version(ctx)
		if err != nil {
			return err
		}

		if cctx.Bool("manage-fdlimit") {
			if _, _, err := ulimit.ManageFdLimit(); err != nil {
				log.Errorf("setting file descriptor limit: %s", err)
			}
		}

		if v.APIVersion != build.APIVersion {
			return xerrors.Errorf("lotus-daemon API version doesn't match: local: %s", api.Version{APIVersion: build.APIVersion})
		}

		log.Info("Checking full node sync status")

		if !cctx.Bool("nosync") {
			if err := lcli.SyncWait(ctx, nodeApi); err != nil {
				return xerrors.Errorf("sync wait: %w", err)
			}
		}

		minerRepoPath := cctx.String(FlagMinerRepo)
		r, err := repo.NewFS(minerRepoPath)
		if err != nil {
			return err
		}

		ok, err := r.Exists()
		if err != nil {
			return err
		}
		if !ok {
			return xerrors.Errorf("repo at '%s' is not initialized, run 'lotus-miner init' to set it up", minerRepoPath)
		}

		// implement by hlm
		// init storage database
		database.InitDB(minerRepoPath)
		// mount nfs storage node
		if err := database.MountAllStorage(false); err != nil {
			return errors.As(err)
		}
		// clean storage cur_work cause by no worker on starting.
		if err := database.ClearStorageWork(); err != nil {
			return errors.As(err)
		}
		defer func() {
			log.Info("RollbackAllStorageTx by program exit.")
			database.RollbackAllStorageTx()
		}()
		// checking sealed for proof
		if err := ffiwrapper.CheckSealed(minerRepoPath); err != nil {
			return errors.As(err)
		}
		// implement by hlm end.

		shutdownChan := make(chan struct{})

		var minerapi api.StorageMiner
		stop, err := node.New(ctx,
			node.StorageMiner(&minerapi),
			node.Override(new(dtypes.ShutdownChan), shutdownChan),
			node.Online(),
			node.Repo(r),

			node.ApplyIf(func(s *node.Settings) bool { return cctx.IsSet("api") },
				node.Override(new(dtypes.APIEndpoint), func() (dtypes.APIEndpoint, error) {
					return multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/" + cctx.String("api"))
				})),
			node.Override(new(api.FullNode), nodeApi),
		)
		if err != nil {
			return err
		}

		endpoint, err := r.APIEndpoint()
		if err != nil {
			return err
		}

		// Bootstrap with full node
		remoteAddrs, err := nodeApi.NetAddrsListen(ctx)
		if err != nil {
			return err
		}

		if err := minerapi.NetConnect(ctx, remoteAddrs); err != nil {
			return err
		}

		log.Infof("Remote version %s", v)

		lst, err := manet.Listen(endpoint)
		if err != nil {
			return xerrors.Errorf("could not listen: %w", err)
		}

		mux := mux.NewRouter()

		rpcServer := jsonrpc.NewServer()
		rpcServer.Register("Filecoin", apistruct.PermissionedStorMinerAPI(minerapi))

		mux.Handle("/rpc/v0", rpcServer)
		mux.PathPrefix("/remote").HandlerFunc(minerapi.(*impl.StorageMinerAPI).ServeRemote)
		mux.PathPrefix("/file").HandlerFunc((&fileserver.FileHandle{Repo: minerRepoPath}).FileHttpServer)
		mux.PathPrefix("/").Handler(http.DefaultServeMux) // pprof

		ah := &auth.Handler{
			Verify: minerapi.AuthVerify,
			Next:   mux.ServeHTTP,
		}

		srv := &http.Server{Handler: ah}

		sigChan := make(chan os.Signal, 2)
		go func() {
			select {
			case <-sigChan:
			case <-shutdownChan:
			}

			log.Warn("Shutting down...")
			if err := stop(context.TODO()); err != nil {
				log.Errorf("graceful shutting down failed: %s", err)
			}
			if err := srv.Shutdown(context.TODO()); err != nil {
				log.Errorf("shutting down RPC server failed: %s", err)
			}
			log.Warn("Graceful shutdown successful")
		}()
		signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

		if reportUrl := cctx.String("report-url"); len(reportUrl) > 0 {
			report.SetReportUrl(reportUrl)
		}
		return srv.Serve(manet.NetListener(lst))
	},
}
