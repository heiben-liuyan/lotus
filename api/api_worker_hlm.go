package api

import (
	"context"

	"github.com/filecoin-project/lotus/build"
	"github.com/filecoin-project/specs-actors/actors/abi"
	"github.com/filecoin-project/specs-storage/storage"
)

type WindowPoStResp struct {
	Proofs []abi.PoStProof
	Ignore []abi.SectorID
}

type WorkerHlmAPI interface {
	Version(context.Context) (build.Version, error)

	SealCommit2(context.Context, abi.SectorID, storage.Commit1Out) (storage.Proof, error)
	GenerateWinningPoSt(context.Context, abi.ActorID, []abi.SectorInfo, abi.PoStRandomness) ([]abi.PoStProof, error)
	GenerateWindowPoSt(context.Context, abi.ActorID, []abi.SectorInfo, abi.PoStRandomness) (WindowPoStResp, error)
}
