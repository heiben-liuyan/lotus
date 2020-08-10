package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/sector-storage/database"
	"github.com/filecoin-project/sector-storage/ffiwrapper"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/filecoin-project/specs-storage/storage"
	sealing "github.com/filecoin-project/storage-fsm"

	"github.com/gwaylib/errors"
)

type worker struct {
	minerEndpoint string
	repo          string
	sealedRepo    string
	auth          http.Header

	actAddr address.Address

	workerSB  *ffiwrapper.Sealer
	rpcServer *rpcServer
	workerCfg ffiwrapper.WorkerCfg

	workMu sync.Mutex
	workOn map[string]ffiwrapper.WorkerTask // task key

	pushMu            sync.Mutex
	sealedMounted     map[string]string
	sealedMountedFile string
}

func acceptJobs(ctx context.Context,
	workerSB, sealedSB *ffiwrapper.Sealer,
	rpcServer *rpcServer,
	act, workerAddr address.Address,
	endpoint string, auth http.Header,
	repo, sealedRepo, mountedFile string,
	workerCfg ffiwrapper.WorkerCfg,
) error {
	api, err := GetNodeApi()
	if err != nil {
		return errors.As(err)
	}
	w := &worker{
		minerEndpoint: endpoint,
		repo:          repo,
		sealedRepo:    sealedRepo,
		auth:          auth,

		actAddr:   act,
		workerSB:  workerSB,
		rpcServer: rpcServer,
		workerCfg: workerCfg,

		workOn: map[string]ffiwrapper.WorkerTask{},

		sealedMounted:     map[string]string{},
		sealedMountedFile: mountedFile,
	}
	tasks, err := api.WorkerQueue(ctx, workerCfg)
	if err != nil {
		return errors.As(err)
	}
	log.Infof("Worker(%s) started", workerCfg.ID)

loop:
	for {
		// log.Infof("Waiting for new task")
		// checking is connection aliveable,if not, do reconnect.
		aliveChecking := time.After(1 * time.Minute) // waiting out
		select {
		case <-aliveChecking:
			ReleaseNodeApi(false)
			_, err := GetNodeApi()
			if err != nil {
				log.Warn(errors.As(err))
			}
		case task := <-tasks:
			if task.SectorID.Miner == 0 {
				// connection is down.
				return errors.New("server shutdown").As(task)
			}

			log.Infof("New task: %s, sector %s, action: %d", task.Key(), task.GetSectorID(), task.Type)
			go func(task ffiwrapper.WorkerTask) {
				taskKey := task.Key()
				w.workMu.Lock()
				if _, ok := w.workOn[taskKey]; ok {
					w.workMu.Unlock()
					// when the miner restart, it should send the same task,
					// and this worker is already working on, so drop this job.
					log.Infof("task(%s) is in working", taskKey)
					return
				} else {
					w.workOn[taskKey] = task
					w.workMu.Unlock()
				}

				defer func() {
					w.workMu.Lock()
					delete(w.workOn, taskKey)
					w.workMu.Unlock()
				}()

				res := w.processTask(ctx, task)
				w.workerDone(ctx, task, res)

				log.Infof("Task %s done, err: %+v", task.Key(), res.GoErr)
			}(task)

		case <-ctx.Done():
			break loop
		}
	}

	log.Warn("acceptJobs exit")
	return nil
}

func (w *worker) addPiece(ctx context.Context, task ffiwrapper.WorkerTask) ([]abi.PieceInfo, error) {
	sizes := task.PieceSizes

	s := sealing.NewSealPiece(w.actAddr, w.workerSB)
	g := &sealing.Pledge{
		SectorID:      task.SectorID,
		Sealing:       s,
		SectorBuilder: w.workerSB,
		ActAddr:       w.actAddr,
		Sizes:         sizes,
	}
	return g.PledgeSector(ctx)
}

func (w *worker) RemoveCache(ctx context.Context, sid string) error {
	w.workMu.Lock()
	defer w.workMu.Unlock()

	if filepath.Base(w.repo) == ".lotusstorage" {
		return nil
	}

	log.Infof("Remove cache:%s,%s", w.repo, sid)
	if err := os.RemoveAll(filepath.Join(w.repo, "sealed", sid)); err != nil {
		log.Error(errors.As(err, sid))
	}
	if err := os.RemoveAll(filepath.Join(w.repo, "cache", sid)); err != nil {
		log.Error(errors.As(err, sid))
	}
	if err := os.RemoveAll(filepath.Join(w.repo, "unsealed", sid)); err != nil {
		log.Error(errors.As(err, sid))
	}
	return nil
}

func (w *worker) CleanCache(ctx context.Context) error {
	w.workMu.Lock()
	defer w.workMu.Unlock()

	// not do this on miner repo
	if filepath.Base(w.repo) == ".lotusstorage" {
		return nil
	}

	sealed := filepath.Join(w.repo, "sealed")
	cache := filepath.Join(w.repo, "cache")
	// staged := filepath.Join(w.repo, "staging")
	unsealed := filepath.Join(w.repo, "unsealed")
	if err := w.cleanCache(ctx, sealed); err != nil {
		return errors.As(err)
	}
	if err := w.cleanCache(ctx, cache); err != nil {
		return errors.As(err)
	}
	if err := w.cleanCache(ctx, unsealed); err != nil {
		return errors.As(err)
	}
	return nil
}

func (w *worker) cleanCache(ctx context.Context, path string) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Warn(errors.As(err))
	} else {
		fileNames := []string{}
		for _, f := range files {
			fileNames = append(fileNames, f.Name())
		}
		api, err := GetNodeApi()
		if err != nil {
			return errors.As(err)
		}
		ws, err := api.WorkerWorkingById(ctx, fileNames)
		if err != nil {
			ReleaseNodeApi(false)
			return errors.As(err, fileNames)
		}
	sealedLoop:
		for _, f := range files {
			for _, s := range ws {
				if s.ID == f.Name() {
					continue sealedLoop
				}
			}
			log.Infof("Remove %s", filepath.Join(path, f.Name()))
			if err := os.RemoveAll(filepath.Join(path, f.Name())); err != nil {
				return errors.As(err, w.workerCfg.IP, filepath.Join(path, f.Name()))
			}
		}
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return errors.As(err, w.workerCfg.IP)
	}
	return nil
}

func (w *worker) mountPush(sid, mountType, mountUri, mountDir, mountOpt string) error {
	// mount
	if err := os.MkdirAll(mountDir, 0755); err != nil {
		return errors.As(err, mountDir)
	}
	w.pushMu.Lock()
	w.sealedMounted[sid] = mountDir
	mountedData, err := json.Marshal(w.sealedMounted)
	if err != nil {
		w.pushMu.Unlock()
		return errors.As(err, w.sealedMountedFile)
	}
	if err := ioutil.WriteFile(w.sealedMountedFile, mountedData, 0666); err != nil {
		w.pushMu.Unlock()
		return errors.As(err, w.sealedMountedFile)
	}
	w.pushMu.Unlock()

	// a fix point, link or mount to the targe file.
	if err := database.Mount(
		mountType,
		mountUri,
		mountDir,
		mountOpt,
	); err != nil {
		return errors.As(err)
	}
	return nil
}

func (w *worker) umountPush(sid, mountDir string) error {
	// umount and client the tmp file
	if _, err := database.Umount(mountDir); err != nil {
		return errors.As(err)
	}
	log.Infof("Remove mount point:%s", mountDir)
	if err := os.RemoveAll(mountDir); err != nil {
		return errors.As(err)
	}

	w.pushMu.Lock()
	delete(w.sealedMounted, sid)
	mountedData, err := json.Marshal(w.sealedMounted)
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(w.sealedMountedFile, mountedData, 0666); err != nil {
		panic(err)
	}
	w.pushMu.Unlock()
	return nil
}

func (w *worker) PushCache(ctx context.Context, task ffiwrapper.WorkerTask) error {
	sid := task.GetSectorID()
	log.Infof("PushCache:%+v", sid)
	defer log.Infof("PushCache exit:%+v", sid)

	api, err := GetNodeApi()
	if err != nil {
		return errors.As(err)
	}
	storage, err := api.PreStorageNode(ctx, sid, w.workerCfg.IP)
	if err != nil {
		return errors.As(err)
	}
	mountUri := storage.MountTransfUri
	if strings.Index(mountUri, w.workerCfg.IP) > -1 {
		log.Infof("found local storage, chagne %s to mount local", mountUri)
		// fix to 127.0.0.1 if it has the same ip.
		mountUri = strings.Replace(mountUri, w.workerCfg.IP, "127.0.0.1", -1)
	}
	mountDir := filepath.Join(w.sealedRepo, sid)
	if err := w.mountPush(
		sid,
		storage.MountType,
		mountUri,
		mountDir,
		storage.MountOpt,
	); err != nil {
		return errors.As(err)
	}
	sealedPath := filepath.Join(mountDir, "sealed")
	if err := os.MkdirAll(sealedPath, 0755); err != nil {
		return errors.As(err)
	}
	// "sealed" is created during previous step
	if err := w.pushRemote(ctx, "sealed", sid, filepath.Join(sealedPath, sid)); err != nil {
		return errors.As(err)
	}
	cachePath := filepath.Join(mountDir, "cache", sid)
	if err := os.MkdirAll(cachePath, 0755); err != nil {
		return errors.As(err)
	}
	if err := w.pushRemote(ctx, "cache", sid, cachePath); err != nil {
		return errors.As(err)
	}
	if err := w.umountPush(sid, mountDir); err != nil {
		return errors.As(err)
	}
	if err := api.CommitStorageNode(ctx, sid); err != nil {
		return errors.As(err)
	}
	return nil
}

func (w *worker) pushCommit(ctx context.Context, task ffiwrapper.WorkerTask) error {
repush:
	select {
	case <-ctx.Done():
		return ffiwrapper.ErrWorkerExit.As(task)
	default:
		// TODO: check cache is support two task
		api, err := GetNodeApi()
		if err != nil {
			log.Warn(errors.As(err))
			goto repush
		}
		// release the worker when pushing happened
		if err := api.WorkerUnlock(ctx, w.workerCfg.ID, task.Key(), "pushing commit", database.SECTOR_STATE_PUSH); err != nil {
			log.Warn(errors.As(err))

			if errors.ErrNoData.Equal(err) {
				// drop data
				return nil
			}

			ReleaseNodeApi(false)
			goto repush
		}

		if err := w.PushCache(ctx, task); err != nil {
			log.Error(errors.As(err, task))
			time.Sleep(60e9)
			goto repush
		}
		if err := w.RemoveCache(ctx, task.GetSectorID()); err != nil {
			log.Warn(errors.As(err))
		}
	}
	return nil
}

func (w *worker) workerDone(ctx context.Context, task ffiwrapper.WorkerTask, res ffiwrapper.SealRes) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			log.Info("Get Node Api")
			api, err := GetNodeApi()
			if err != nil {
				log.Warn(errors.As(err))
				continue
			}
			log.Info("Do WorkerDone")
			if err := api.WorkerDone(ctx, res); err != nil {
				if errors.ErrNoData.Equal(err) {
					log.Warn("caller not found, drop this task:%+v", task)
					return
				}

				log.Warn(errors.As(err))

				ReleaseNodeApi(false)
				continue
			}

			// pass
			return

		}
	}
}

func (w *worker) processTask(ctx context.Context, task ffiwrapper.WorkerTask) ffiwrapper.SealRes {
	res := ffiwrapper.SealRes{
		Type:      task.Type,
		TaskID:    task.Key(),
		WorkerCfg: w.workerCfg,
	}

	switch task.Type {
	case ffiwrapper.WorkerAddPiece:
	case ffiwrapper.WorkerPreCommit1:
	case ffiwrapper.WorkerPreCommit2:
	case ffiwrapper.WorkerCommit1:
	case ffiwrapper.WorkerCommit2:
	case ffiwrapper.WorkerFinalize:
	case ffiwrapper.WorkerWindowPoSt:
		proofs, err := w.rpcServer.GenerateWindowPoSt(ctx,
			task.SectorID.Miner,
			task.SectorInfo,
			task.Randomness,
		)
		if err != nil {
			return errRes(errors.As(err, w.workerCfg), task)
		}
		res.WindowPoStProofOut = proofs.Proofs
		res.WindowPoStIgnSectors = proofs.Ignore
		return res
	case ffiwrapper.WorkerWinningPoSt:
		proofs, err := w.rpcServer.GenerateWinningPoSt(ctx,
			task.SectorID.Miner,
			task.SectorInfo,
			task.Randomness,
		)
		if err != nil {
			return errRes(errors.As(err, w.workerCfg), task)
		}
		res.WinningPoStProofOut = proofs
		return res
	default:
		return errRes(errors.New("unknown task type").As(task.Type, w.workerCfg), task)
	}
	api, err := GetNodeApi()
	if err != nil {
		ReleaseNodeApi(false)
		return errRes(errors.As(err, w.workerCfg), task)
	}
	// clean cache before working.
	if err := w.CleanCache(ctx); err != nil {
		return errRes(errors.As(err, w.workerCfg), task)
	}
	// checking is the cache in a different storage server, do fetch when it is.
	if w.workerCfg.CacheMode == 0 &&
		task.Type > ffiwrapper.WorkerAddPiece && task.Type < ffiwrapper.WorkerCommit2 &&
		task.WorkerID != w.workerCfg.ID {
		// lock bandwidth
		if err := api.WorkerAddConn(ctx, task.WorkerID, 1); err != nil {
			ReleaseNodeApi(false)
			return errRes(errors.As(err, w.workerCfg), task)
		}
	retryFetch:
		// fetch data
		uri := task.SectorStorage.WorkerInfo.SvcUri
		if len(uri) == 0 {
			uri = w.minerEndpoint
		}
		if err := w.fetchRemote(
			"http://"+uri,
			task.SectorStorage.SectorInfo.ID,
			task.Type,
		); err != nil {
			log.Warnf("fileserver error, retry 10s later:%+s", err.Error())
			time.Sleep(10e9)
			goto retryFetch
		}
		// release bandwidth
		if err := api.WorkerAddConn(ctx, task.WorkerID, -1); err != nil {
			ReleaseNodeApi(false)
			return errRes(errors.As(err, w.workerCfg), task)
		}
		// release the storage cache
		log.Infof("fetch %s done, try delete remote files.", task.Key())
		if err := w.deleteRemoteCache(
			"http://"+uri,
			task.SectorStorage.SectorInfo.ID,
		); err != nil {
			return errRes(errors.As(err, w.workerCfg), task)
		}
	}
	// lock the task to this worker
	if err := api.WorkerLock(ctx, w.workerCfg.ID, task.Key(), "task in", int(task.Type)); err != nil {
		ReleaseNodeApi(false)
		return errRes(errors.As(err, w.workerCfg), task)
	}
	unlockWorker := false
	switch task.Type {
	case ffiwrapper.WorkerAddPiece:
		rsp, err := w.addPiece(ctx, task)
		if err != nil {
			return errRes(errors.As(err, w.workerCfg), task)
		}
		res.Pieces = rsp

		// checking is the next step interrupted
		unlockWorker = (w.workerCfg.ParallelPrecommit1 == 0)

	case ffiwrapper.WorkerPreCommit1:
		pieceInfo, err := ffiwrapper.DecodePieceInfo(task.Pieces)
		if err != nil {
			return errRes(errors.As(err, w.workerCfg), task)
		}
		rspco, err := w.workerSB.SealPreCommit1(ctx, task.SectorID, task.SealTicket, pieceInfo)
		if err != nil {
			return errRes(errors.As(err, w.workerCfg), task)
		}
		res.PreCommit1Out = rspco

		// checking is the next step interrupted
		unlockWorker = (w.workerCfg.ParallelPrecommit2 == 0)
	case ffiwrapper.WorkerPreCommit2:
		out, err := w.workerSB.SealPreCommit2(ctx, task.SectorID, task.PreCommit1Out)
		if err != nil {
			return errRes(errors.As(err, w.workerCfg), task)
		}
		res.PreCommit2Out = ffiwrapper.SectorCids{
			Unsealed: out.Unsealed.String(),
			Sealed:   out.Sealed.String(),
		}
	case ffiwrapper.WorkerCommit1:
		pieceInfo, err := ffiwrapper.DecodePieceInfo(task.Pieces)
		if err != nil {
			return errRes(errors.As(err, w.workerCfg), task)
		}
		cids, err := task.Cids.Decode()
		if err != nil {
			return errRes(errors.As(err, w.workerCfg), task)
		}
		out, err := w.workerSB.SealCommit1(ctx, task.SectorID, task.SealTicket, task.SealSeed, pieceInfo, *cids)
		if err != nil {
			return errRes(errors.As(err, w.workerCfg), task)
		}
		res.Commit1Out = out
	case ffiwrapper.WorkerCommit2:
		var out storage.Proof
		var err error
		// if local no gpu service, using remote if the remtoes have.
		// TODO: Optimized waiting algorithm
		if w.workerCfg.ParallelCommit2 == 0 && !w.workerCfg.Commit2Srv {
			for {
				out, err = CallCommit2Service(ctx, task)
				if err != nil {
					log.Warn(errors.As(err))
					time.Sleep(10e9)
					continue
				}
				break
			}
		}
		// call gpu service failed, using local instead.
		if len(out) == 0 {
			out, err = w.workerSB.SealCommit2(ctx, task.SectorID, task.Commit1Out)
			if err != nil {
				return errRes(errors.As(err, w.workerCfg), task)
			}
		}
		res.Commit2Out = out
	// SPEC: cancel deal with worker finalize, because it will post failed when commit2 is online and finalize is interrupt.
	// SPEC: maybe it should failed on commit2 but can not failed on transfering the finalize data on windowpost.
	// TODO: when testing stable finalize retrying and reopen it.
	case ffiwrapper.WorkerFinalize:
		sealedFile := w.workerSB.SectorPath("sealed", task.GetSectorID())
		_, err := os.Stat(string(sealedFile))
		if err != nil {
			if !os.IsNotExist(err) {
				return errRes(errors.As(err, sealedFile), task)
			}
		} else {
			if err := w.workerSB.FinalizeSector(ctx, task.SectorID, nil); err != nil {
				return errRes(errors.As(err, w.workerCfg), task)
			}
			if err := w.pushCommit(ctx, task); err != nil {
				return errRes(errors.As(err, w.workerCfg), task)
			}
		}
	}

	// release the worker when stage is interrupted
	if unlockWorker {
		log.Info("Release Worker by:", task)
		if err := api.WorkerUnlock(ctx, w.workerCfg.ID, task.Key(), "transfer to another worker", database.SECTOR_STATE_MOVE); err != nil {
			log.Warn(errors.As(err))
			ReleaseNodeApi(false)
			return errRes(errors.As(err, w.workerCfg), task)
		}
	}
	return res
}

func errRes(err error, task ffiwrapper.WorkerTask) ffiwrapper.SealRes {
	return ffiwrapper.SealRes{
		Type:   task.Type,
		TaskID: task.Key(),
		Err:    err.Error(),
		GoErr:  err,
		WorkerCfg: ffiwrapper.WorkerCfg{
			//
			ID: task.WorkerID,
		},
	}
}
