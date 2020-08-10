package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/filecoin-project/go-jsonrpc"
	paramfetch "github.com/filecoin-project/go-paramfetch"
	"github.com/filecoin-project/sector-storage/database"
	"github.com/filecoin-project/sector-storage/ffiwrapper"
	"github.com/filecoin-project/sector-storage/ffiwrapper/basicfs"
	mux "github.com/gorilla/mux"
	"github.com/mitchellh/go-homedir"

	"github.com/filecoin-project/go-jsonrpc/auth"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"golang.org/x/xerrors"

	manet "github.com/multiformats/go-multiaddr-net"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/apistruct"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/lotus/chain/types"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/lotus/lib/fileserver"
	"github.com/filecoin-project/lotus/lib/lotuslog"
	"github.com/filecoin-project/lotus/lib/report"
	"github.com/filecoin-project/lotus/node/repo"

	"github.com/gwaylib/errors"
)

var log = logging.Logger("main")

var (
	nodeApi    api.StorageMiner
	nodeCloser jsonrpc.ClientCloser
	nodeCCtx   *cli.Context
	nodeSync   sync.Mutex
)

func closeNodeApi() {
	if nodeCloser != nil {
		nodeCloser()
	}
	nodeApi = nil
	nodeCloser = nil
}

func ReleaseNodeApi(shutdown bool) {
	nodeSync.Lock()
	defer nodeSync.Unlock()
	time.Sleep(3e9)

	if nodeApi == nil {
		return
	}

	if shutdown {
		closeNodeApi()
		return
	}

	ctx := lcli.ReqContext(nodeCCtx)

	// try reconnection
	_, err := nodeApi.Version(ctx)
	if err != nil {
		closeNodeApi()
		return
	}
}

func GetNodeApi() (api.StorageMiner, error) {
	nodeSync.Lock()
	defer nodeSync.Unlock()

	if nodeApi != nil {
		return nodeApi, nil
	}

	ctx := lcli.ReqContext(nodeCCtx)
	if nodeCCtx == nil {
		panic("need init node cctx")
	}

	nApi, closer, err := lcli.GetStorageMinerAPI(nodeCCtx)
	if err != nil {
		closeNodeApi()
		return nil, errors.As(err)
	}
	nodeApi = nApi
	nodeCloser = closer

	v, err := nodeApi.Version(ctx)
	if err != nil {
		closeNodeApi()
		return nil, errors.As(err)
	}
	if v.APIVersion != build.APIVersion {
		closeNodeApi()
		return nil, xerrors.Errorf("lotus-storage-miner API version doesn't match: local: ", api.Version{APIVersion: build.APIVersion})
	}

	return nodeApi, nil
}

func AuthVerify(ctx context.Context, token string) ([]auth.Permission, error) {
	nApi, err := GetNodeApi()
	if err != nil {
		return nil, errors.As(err)
	}
	return nApi.AuthVerify(ctx, token)
}

func main() {
	lotuslog.SetupLogLevels()

	log.Info("Starting lotus worker")

	local := []*cli.Command{
		runCmd,
	}

	app := &cli.App{
		Name:    "lotus-seal-worker",
		Usage:   "Remote storage miner worker",
		Version: build.UserVersion(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "repo",
				EnvVars: []string{"LOTUS_WORKER_PATH", "WORKER_PATH"},
				Value:   "~/.lotusstorage", // TODO: Consider XDG_DATA_HOME
			},
			&cli.StringFlag{
				Name:    "miner-repo",
				EnvVars: []string{"LOTUS_MINER_PATH", "LOTUS_STORAGE_PATH"},
				Value:   "~/.lotusstorage", // TODO: Consider XDG_DATA_HOME
			},
			&cli.StringFlag{
				Name:    "storage-repo",
				EnvVars: []string{"WORKER_SEALED_PATH"},
				Value:   "~/.lotusworker/push", // TODO: Consider XDG_DATA_HOME
			},
			&cli.StringFlag{
				Name:    "id-file",
				EnvVars: []string{"WORKER_ID_PATH"},
				Value:   "~/.lotusworker/worker.id", // TODO: Consider XDG_DATA_HOME
			},
			&cli.StringFlag{
				Name:  "report-url",
				Value: "",
				Usage: "report url for state",
			},
			&cli.StringFlag{
				Name:  "listen-addr",
				Value: "",
				Usage: "listen address a local",
			},
			&cli.StringFlag{
				Name:  "server-addr",
				Value: "",
				Usage: "server address for visit",
			},
			&cli.UintFlag{
				Name:  "max-tasks",
				Value: 1,
				Usage: "max tasks for memory.",
			},
			&cli.UintFlag{
				Name:  "transfer-buffer",
				Value: 1,
				Usage: "buffer for transfer when task done",
			},
			&cli.UintFlag{
				Name:  "cache-mode",
				Value: 0,
				Usage: "cache mode.0, tranfer mode; 1, share mode",
			},
			&cli.UintFlag{
				Name:  "parallel-addpiece",
				Value: 1,
				Usage: "Parallel of addpice, <= max-tasks",
			},
			&cli.UintFlag{
				Name:  "parallel-precommit1",
				Value: 1,
				Usage: "Parallel of precommit1, <= max-tasks",
			},
			&cli.UintFlag{
				Name:  "parallel-precommit2",
				Value: 1,
				Usage: "Parallel of precommit2, <= max-tasks",
			},
			&cli.UintFlag{
				Name:  "parallel-commit1",
				Value: 1,
				Usage: "Parallel of commit1,0< parallel <= max-tasks, undefined for 0",
			},
			&cli.UintFlag{
				Name:  "parallel-commit2",
				Value: 1,
				Usage: "Parallel of commit2, <= max-tasks. if parallel is 0, will select a commit2 service until success",
			},
			&cli.BoolFlag{
				Name:  "commit2-srv",
				Value: false,
				Usage: "Open commit2 service, need parallel-commit2 > 0",
			},
			&cli.BoolFlag{
				Name:  "wdpost-srv",
				Value: false,
				Usage: "Open window PoSt service",
			},
			&cli.BoolFlag{
				Name:  "wnpost-srv",
				Value: false,
				Usage: "Open wining PoSt service",
			},
		},

		Commands: local,
	}
	app.Setup()
	app.Metadata["repoType"] = repo.StorageMiner

	if err := app.Run(os.Args); err != nil {
		log.Warnf("%+v", err)
		return
	}
}

var runCmd = &cli.Command{
	Name:  "run",
	Usage: "Start lotus worker",
	Action: func(cctx *cli.Context) error {
		nodeCCtx = cctx

		nodeApi, err := GetNodeApi()
		if err != nil {
			return xerrors.Errorf("getting miner api: %w", err)
		}
		defer ReleaseNodeApi(true)

		ainfo, err := lcli.GetAPIInfo(cctx, repo.StorageMiner)
		if err != nil {
			return xerrors.Errorf("could not get api info: %w", err)
		}
		_, storageAddr, err := manet.DialArgs(ainfo.Addr)
		if err != nil {
			return err
		}

		r, err := homedir.Expand(cctx.String("repo"))
		if err != nil {
			return err
		}

		minerRepo, err := homedir.Expand(cctx.String("miner-repo"))
		if err != nil {
			return err
		}

		sealedRepo, err := homedir.Expand(cctx.String("storage-repo"))
		if err != nil {
			return err
		}
		idFile := cctx.String("id-file")
		if len(idFile) == 0 {
			idFile = "~/.lotusworker/worker.id"
		}
		workerIdFile, err := homedir.Expand(idFile)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithCancel(lcli.ReqContext(nodeCCtx))
		defer cancel()

		act, err := nodeApi.ActorAddress(ctx)
		if err != nil {
			return err
		}
		ssize, err := nodeApi.ActorSectorSize(ctx, act)
		if err != nil {
			return err
		}
		log.Infof("Running ActorSize:%s", ssize.ShortString())

		workerAddr, err := nodeApi.WorkerAddress(ctx, act, types.EmptyTSK)
		if err != nil {
			return errors.As(err)
		}

		// fetch parameters from hlm
		// TODO: need more develop time
		//if err := FetchHlmParams(ctx, nodeApi, storageAddr); err != nil {
		//	return errors.As(err)
		//}

		if err := paramfetch.GetParams(ctx, build.ParametersJSON(), uint64(ssize)); err != nil {
			return xerrors.Errorf("get params: %w", err)
		}

		if err := os.MkdirAll(r, 0755); err != nil {
			return errors.As(err, r)
		}
		spt, err := ffiwrapper.SealProofTypeFromSectorSize(ssize)
		if err != nil {
			return xerrors.Errorf("getting proof type: %w", err)
		}
		cfg := &ffiwrapper.Config{
			SealProofType: spt,
		}

		workerSealer, err := ffiwrapper.New(ffiwrapper.RemoteCfg{}, &basicfs.Provider{
			Root: r,
		}, cfg)
		if err != nil {
			return err
		}
		minerSealer, err := ffiwrapper.New(ffiwrapper.RemoteCfg{}, &basicfs.Provider{
			Root: minerRepo,
		}, cfg)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(sealedRepo, 0755); err != nil {
			return errors.As(err, sealedRepo)
		}
		sealedSB, err := ffiwrapper.New(ffiwrapper.RemoteCfg{}, &basicfs.Provider{
			Root: sealedRepo,
		}, cfg)
		if err != nil {
			return errors.As(err, sealedRepo)
		}
		workerId := GetWorkerID(workerIdFile)
		netIp := os.Getenv("NETIP")
		// make serivce server
		listenAddr := cctx.String("listen-addr")
		if len(listenAddr) == 0 {
			listenAddr = netIp + ":1280"
		}
		serverAddr := cctx.String("server-addr")
		if len(serverAddr) == 0 {
			serverAddr = listenAddr
		}

		// init and clean the sealed dir who has mounted.
		sealedMountedFile := workerIdFile + ".lock"
		sealedMounted := map[string]string{}
		if mountedData, err := ioutil.ReadFile(sealedMountedFile); err == nil {
			if err := json.Unmarshal(mountedData, &sealedMounted); err != nil {
				return errors.As(err)
			}
			for _, p := range sealedMounted {
				if _, err := database.Umount(p); err != nil {
					log.Info(err)
				} else {
					if err := os.RemoveAll(p); err != nil {
						log.Error(err)
					}
				}
			}
		}

		// init worker configuration
		workerCfg := ffiwrapper.WorkerCfg{
			ID:                 workerId,
			IP:                 netIp,
			SvcUri:             serverAddr,
			MaxTaskNum:         int(cctx.Uint("max-tasks")),
			CacheMode:          int(cctx.Uint("cache-mode")),
			TransferBuffer:     int(cctx.Uint("transfer-buffer")),
			ParallelAddPiece:   int(cctx.Uint("parallel-addpiece")),
			ParallelPrecommit1: int(cctx.Uint("parallel-precommit1")),
			ParallelPrecommit2: int(cctx.Uint("parallel-precommit2")),
			ParallelCommit1:    int(cctx.Uint("parallel-commit1")),
			ParallelCommit2:    int(cctx.Uint("parallel-commit2")),
			Commit2Srv:         cctx.Bool("commit2-srv"),
			WdPoStSrv:          cctx.Bool("wdpost-srv"),
			WnPoStSrv:          cctx.Bool("wnpost-srv"),
		}
		workerApi := &rpcServer{
			sb:           minerSealer,
			storageCache: map[int64]database.StorageInfo{},
		}
		if workerCfg.WdPoStSrv || workerCfg.WdPoStSrv {
			if err := workerApi.loadMinerStorage(ctx); err != nil {
				return errors.As(err)
			}
		}

		mux := mux.NewRouter()
		rpcServer := jsonrpc.NewServer()
		rpcServer.Register("Filecoin", apistruct.PermissionedWorkerHlmAPI(workerApi))
		mux.Handle("/rpc/v0", rpcServer)
		mux.PathPrefix("/file").HandlerFunc((&fileserver.FileHandle{Repo: r}).FileHttpServer)
		mux.PathPrefix("/").Handler(http.DefaultServeMux) // pprof

		ah := &auth.Handler{
			Verify: AuthVerify,
			Next:   mux.ServeHTTP,
		}
		srv := &http.Server{
			Handler: ah,
			BaseContext: func(listener net.Listener) context.Context {
				return ctx
			},
		}
		nl, err := net.Listen("tcp", listenAddr)
		if err != nil {
			return err
		}

		go func() {
			if err := srv.Serve(nl); err != nil {
				log.Error(err)
				os.Exit(1)
			}
		}()

		// set report url
		if reportUrl := cctx.String("report-url"); len(reportUrl) > 0 {
			report.SetReportUrl(reportUrl)
		}

		if err := acceptJobs(ctx,
			workerSealer, sealedSB,
			workerApi,
			act, workerAddr,
			storageAddr, ainfo.AuthHeader(),
			r, sealedRepo, sealedMountedFile,
			workerCfg,
		); err != nil {
			log.Warn(err)
			ReleaseNodeApi(false)
		}
		return nil
	},
}
