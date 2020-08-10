package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
)

var (
	workerid   = ""
	workeridMu = sync.Mutex{}
)

func GetWorkerID(file string) string {
	workeridMu.Lock()
	defer workeridMu.Unlock()

	if len(workerid) > 0 {
		return workerid
	}
	// checkfile
	id, err := ioutil.ReadFile(file)
	if err == nil {
		workerid = strings.TrimSpace(string(id))
		return workerid
	}
	if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		panic(err)
	}
	workerid = uuid.New().String()
	if err := ioutil.WriteFile(file, []byte(workerid), 0666); err != nil {
		// depend on this
		panic(err)
	}

	return workerid
}
