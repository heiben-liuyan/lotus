package impl

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/lib/fileserver"
	"github.com/filecoin-project/sector-storage/database"
	"github.com/filecoin-project/sector-storage/ffiwrapper"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/filecoin-project/specs-storage/storage"

	"github.com/gwaylib/errors"
)

func (sm *StorageMinerAPI) Testing(ctx context.Context, fnName string, args []string) error {
	return sm.Miner.Testing(ctx, fnName, args)
}

func (sm *StorageMinerAPI) RunPledgeSector(ctx context.Context) error {
	return sm.Miner.RunPledgeSector()
}
func (sm *StorageMinerAPI) StatusPledgeSector(ctx context.Context) (int, error) {
	return sm.Miner.StatusPledgeSector()
}
func (sm *StorageMinerAPI) StopPledgeSector(ctx context.Context) error {
	return sm.Miner.ExitPledgeSector()
}

func (sm *StorageMinerAPI) HlmSectorSetState(ctx context.Context, sid, memo string, state int) error {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).UpdateSectorState(sid, memo, state)
}

func (sm *StorageMinerAPI) HlmSectorFinalize(ctx context.Context, sid string) error {
	id, err := ffiwrapper.ParseSectorID(sid)
	if err != nil {
		return errors.As(err, sid)
	}
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).FinalizeSector(ctx, id, []storage.Range{})
}

// Message communication
func (sm *StorageMinerAPI) HlmSectorListAll(ctx context.Context) ([]api.SectorInfo, error) {
	sectors, err := sm.Miner.ListSectors()
	if err != nil {
		return nil, err
	}

	out := []api.SectorInfo{}
	for _, sector := range sectors {
		out = append(out, api.SectorInfo{
			State:    api.SectorState(sector.State),
			SectorID: sector.SectorNumber,
			// TODO: more?
		})
	}
	return out, nil
}

func (sm *StorageMinerAPI) SelectCommit2Service(ctx context.Context, sector abi.SectorID) (*ffiwrapper.WorkerCfg, error) {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).SelectCommit2Service(ctx, sector)
}

func (sm *StorageMinerAPI) UnlockGPUService(ctx context.Context, workerId, taskKey string) error {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).UnlockGPUService(ctx, workerId, taskKey)
}

func (sm *StorageMinerAPI) WorkerAddress(ctx context.Context, act address.Address, task types.TipSetKey) (address.Address, error) {
	mInfo, err := sm.Full.StateMinerInfo(ctx, act, task)
	if err != nil {
		return address.Address{}, err
	}
	return mInfo.Worker, nil
}

func (sm *StorageMinerAPI) WorkerStatus(ctx context.Context) (ffiwrapper.WorkerStats, error) {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).WorkerStats(), nil
}
func (sm *StorageMinerAPI) WorkerStatusAll(ctx context.Context) ([]ffiwrapper.WorkerRemoteStats, error) {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).WorkerRemoteStats()
}
func (sm *StorageMinerAPI) WorkerQueue(ctx context.Context, cfg ffiwrapper.WorkerCfg) (<-chan ffiwrapper.WorkerTask, error) {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).AddWorker(ctx, cfg)
}
func (sm *StorageMinerAPI) WorkerWorking(ctx context.Context, workerId string) (database.WorkingSectors, error) {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).TaskWorking(workerId)
}
func (sm *StorageMinerAPI) WorkerWorkingById(ctx context.Context, sid []string) (database.WorkingSectors, error) {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).TaskWorkingById(sid)
}
func (sm *StorageMinerAPI) WorkerLock(ctx context.Context, workerId, taskKey, memo string, sectorState int) error {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).LockWorker(ctx, workerId, taskKey, memo, sectorState)
}
func (sm *StorageMinerAPI) WorkerUnlock(ctx context.Context, workerId, taskKey, memo string, sectorState int) error {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).UnlockWorker(ctx, workerId, taskKey, memo, sectorState)
}
func (sm *StorageMinerAPI) WorkerDone(ctx context.Context, res ffiwrapper.SealRes) error {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).TaskDone(ctx, res)
}
func (sm *StorageMinerAPI) WorkerDisable(ctx context.Context, wid string, disable bool) error {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).DisableWorker(ctx, wid, disable)
}
func (sm *StorageMinerAPI) WorkerAddConn(ctx context.Context, wid string, num int) error {
	return database.AddWorkerConn(wid, num)
}
func (sm *StorageMinerAPI) WorkerPreConn(ctx context.Context) (*database.WorkerInfo, error) {
	return database.PrepareWorkerConn()
}
func (sm *StorageMinerAPI) WorkerMinerConn(ctx context.Context) (int, error) {
	return fileserver.Conns(), nil
}
func (sm *StorageMinerAPI) AddHLMStorage(ctx context.Context, sInfo database.StorageInfo) error {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).AddStorage(ctx, sInfo)
}
func (sm *StorageMinerAPI) DisableHLMStorage(ctx context.Context, id int64) error {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).DisableStorage(ctx, id)
}
func (sm *StorageMinerAPI) MountHLMStorage(ctx context.Context, id int64) error {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).MountStorage(ctx, id)
}

func (sm *StorageMinerAPI) UMountHLMStorage(ctx context.Context, id int64) error {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).UMountStorage(ctx, id)
}

func (sm *StorageMinerAPI) RelinkHLMStorage(ctx context.Context, id int64) error {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).RelinkStorage(ctx, id)
}
func (sm *StorageMinerAPI) ReplaceHLMStorage(ctx context.Context, id int64, signalUri, transfUri, mountType, mountOpt string) error {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).ReplaceStorage(ctx, id, signalUri, transfUri, mountType, mountOpt)
}
func (sm *StorageMinerAPI) ScaleHLMStorage(ctx context.Context, id int64, size int64, work int64) error {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).ScaleStorage(ctx, id, size, work)
}
func (sm *StorageMinerAPI) PreStorageNode(ctx context.Context, sectorId, clientIp string) (*database.StorageInfo, error) {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).PreStorageNode(sectorId, clientIp)
}
func (sm *StorageMinerAPI) CommitStorageNode(ctx context.Context, sectorId string) error {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).CommitStorageNode(sectorId)
}
func (sm *StorageMinerAPI) CancelStorageNode(ctx context.Context, sectorId string) error {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).CancelStorageNode(sectorId)
}
func (sm *StorageMinerAPI) ChecksumStorage(ctx context.Context, ver int64) (database.StorageList, error) {
	return sm.StorageMgr.Prover.(*ffiwrapper.Sealer).ChecksumStorage(ver)
}
