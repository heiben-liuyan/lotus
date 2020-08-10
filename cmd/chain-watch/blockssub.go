package main

import (
	"context"
	"encoding/json"
	"io"

	aapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/gwaylib/errors"
)

type BlockHeader struct {
	*types.BlockHeader
	Params interface{} // decode for cid
}

func subBlocks(ctx context.Context, api aapi.FullNode, storage io.Writer) {
	sub, err := api.SyncIncomingBlocks(ctx)
	if err != nil {
		log.Error(err)
		return
	}

	for bh := range sub {
		bhData, err := json.Marshal(bh)
		if err != nil {
			log.Warn(errors.As(err))
		}
		_ = bhData
		// log.Infof("Get subBlocks:%s", string(bhData))
	}
}
