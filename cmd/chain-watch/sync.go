package main

import (
	"context"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/store"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/abi"
)

func runSyncer(ctx context.Context, api api.FullNode) {
	// checking process
	log.Info("Get sync state")
	base, err := api.SyncState(ctx)
	if err != nil {
		log.Error(err)
		return
	}
	if len(base.ActiveSyncs) > 0 {
		log.Infof("Base sync state:%+v", base.ActiveSyncs[0].Base.Height())
		historyHeight, err := GetCurHeight()
		if err != nil {
			log.Error(err)
			return
		}
		baseHeight := int64(base.ActiveSyncs[0].Base.Height())
		for i := historyHeight + 1; i < baseHeight; i++ {
			oldTs, err := api.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(i), types.EmptyTSK)
			if err != nil {
				log.Error(err)
				return
			}
			syncHead(ctx, api, oldTs)
		}
	}

	// listen change
	notifs, err := api.ChainNotify(ctx)
	if err != nil {
		panic(err)
	}
	go func() {
		for notif := range notifs {
			for _, change := range notif {
				switch change.Type {
				case store.HCCurrent:
					fallthrough
				case store.HCApply:
					syncHead(ctx, api, change.Val)
				case store.HCRevert:
					log.Warnf("revert todo")
				}
				log.Info("=====message======", change.Type, ":", store.HCCurrent)
				if change.Type == store.HCCurrent {
					// go subMpool(ctx, api, st, change.Val)
					// go subBlocks(ctx, api, st)
				}
			}
		}
	}()
}
