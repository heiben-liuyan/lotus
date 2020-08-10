package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	//"path/filepath"

	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/lib/fileserver"
	"github.com/gwaylib/errors"
)

func FetchHlmParams(ctx context.Context, napi api.StorageMiner, endpoint string) error {
	paramUri := ""
	// try download from worker
	dlWorker, err := napi.WorkerPreConn(ctx)
	if err != nil {
		if !errors.ErrNoData.Equal(err) {
			return errors.As(err)
		}
		// pass, using miner's
	} else {
		if dlWorker.SvcConn < 2 {
			paramUri = dlWorker.SvcUri
		}
		// else using miner's
	}
	// try download from miner
	if len(paramUri) == 0 {
		minerConns, err := napi.WorkerMinerConn(ctx)
		if err != nil {
			return errors.As(err)
		}
		// no worker online, get from miner
		if minerConns > 10 {
			return errors.New("miner download connections full")
		}
		paramUri = "http://" + endpoint
	}

	for {
		log.Info("try fetch hlm params")
		//		if err := fetchParams(paramUri, "/var/tmp/filecoin-proof-parameters"); err != nil {
		//			log.Warn(errors.As(err))
		//			time.Sleep(10e9)
		//			continue
		//		}
		return nil
	}
}

func fetchParams(serverUri, to string) error {
	var err error
	for i := 0; i < 3; i++ {
		err = tryFetchParams(serverUri, to)
		if err != nil {
			log.Warn(errors.As(err, serverUri, to))
			continue
		}
		return nil
	}
	return err
}
func tryFetchParams(serverUri, to string) error {
	// fetch cache
	cacheResp, err := http.Get(fmt.Sprintf("%s/filecoin-proof-parameters/", serverUri))
	if err != nil {
		return errors.As(err)
	}
	defer cacheResp.Body.Close()
	if cacheResp.StatusCode != 200 {
		return errors.New(cacheResp.Status).As(serverUri)
	}
	cacheRespData, err := ioutil.ReadAll(cacheResp.Body)
	if err != nil {
		return errors.As(err)
	}
	cacheDir := &fileserver.StorageDirectoryResp{}
	if err := xml.Unmarshal(cacheRespData, cacheDir); err != nil {
		return errors.As(err)
	}
	if err := os.MkdirAll(to, 0755); err != nil {
		return errors.As(err)
	}
	//for _, file := range cacheDir.Files {
	//		if err := fetchFile(
	//			fmt.Sprintf("%s/filecoin-proof-parameters/%s", serverUri, file.Value),
	//			filepath.Join(to, file.Value),
	//		); err != nil {
	//			return errors.As(err)
	//		}
	//}
	return nil
}
