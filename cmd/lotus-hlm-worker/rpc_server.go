package main

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/sector-storage/database"
	"github.com/filecoin-project/sector-storage/ffiwrapper"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/filecoin-project/specs-storage/storage"

	"github.com/gwaylib/errors"
)

type rpcServer struct {
	sb *ffiwrapper.Sealer

	storageLk    sync.Mutex
	storageVer   int64
	storageCache map[int64]database.StorageInfo
}

func (w *rpcServer) Version(context.Context) (build.Version, error) {
	return build.APIVersion, nil
}

func (w *rpcServer) SealCommit2(ctx context.Context, sector abi.SectorID, commit1Out storage.Commit1Out) (storage.Proof, error) {
	log.Infof("SealCommit2 RPC in:%d", sector)
	defer log.Infof("SealCommit2 RPC out:%d", sector)

	return w.sb.SealCommit2(ctx, sector, commit1Out)
}

func (w *rpcServer) loadMinerStorage(ctx context.Context) error {
	w.storageLk.Lock()
	defer w.storageLk.Unlock()

	// checksum
	napi, err := GetNodeApi()
	if err != nil {
		return errors.As(err)
	}
	list, err := napi.ChecksumStorage(ctx, w.storageVer)
	if err != nil {
		return errors.As(err)
	}
	// no storage to mount
	if len(list) == 0 {
		return nil
	}

	sumVer := int64(0)
	// mount storage data
	for _, info := range list {
		sumVer += info.Version
		cacheInfo, ok := w.storageCache[info.ID]
		if ok && cacheInfo.Version == info.Version {
			continue
		}

		// version not match
		if err := database.Mount(
			info.MountType,
			info.MountSignalUri,
			filepath.Join(info.MountDir, fmt.Sprintf("%d", info.ID)),
			info.MountOpt,
		); err != nil {
			return errors.As(err)
		}
		w.storageCache[info.ID] = info
	}
	w.storageVer = sumVer

	return nil
}

func (w *rpcServer) GenerateWinningPoSt(ctx context.Context, minerID abi.ActorID, sectorInfo []abi.SectorInfo, randomness abi.PoStRandomness) ([]abi.PoStProof, error) {
	log.Infof("GenerateWinningPoSt RPC in:%d", minerID)
	defer log.Infof("GenerateWinningPoSt RPC out:%d", minerID)

	// load miner storage if not exist
	if err := w.loadMinerStorage(ctx); err != nil {
		return nil, errors.As(err)
	}

	return w.sb.GenerateWinningPoSt(ctx, minerID, sectorInfo, randomness)
}
func (w *rpcServer) GenerateWindowPoSt(ctx context.Context, minerID abi.ActorID, sectorInfo []abi.SectorInfo, randomness abi.PoStRandomness) (api.WindowPoStResp, error) {
	log.Infof("GenerateWindowPoSt RPC in:%d", minerID)
	defer log.Infof("GenerateWindowPoSt RPC out:%d", minerID)

	// load miner storage if not exist
	if err := w.loadMinerStorage(ctx); err != nil {
		return api.WindowPoStResp{}, errors.As(err)
	}

	proofs, ignore, err := w.sb.GenerateWindowPoSt(ctx, minerID, sectorInfo, randomness)
	if err != nil {
		return api.WindowPoStResp{}, errors.As(err)
	}
	return api.WindowPoStResp{
		Proofs: proofs,
		Ignore: ignore,
	}, nil
}
