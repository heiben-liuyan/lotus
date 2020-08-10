package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gwaylib/errors"
)

const (
	append_file_new      = 0
	append_file_continue = 1
	append_file_complete = 2
)

func canAppendFile(aFile, bFile *os.File, aStat, bStat os.FileInfo) (int, error) {
	checksumSize := int64(32 * 1024)
	// for small size, just do rewrite.
	aSize := aStat.Size()
	bSize := bStat.Size()
	if bSize < checksumSize {
		return append_file_new, nil
	}
	if bSize > aSize {
		return append_file_new, nil
	}

	aData := make([]byte, checksumSize)
	bData := make([]byte, checksumSize)
	// TODO: get random data
	if _, err := aFile.ReadAt(aData, bSize-checksumSize); err != nil {
		return append_file_new, errors.As(err)
	}
	if _, err := bFile.ReadAt(bData, bSize-checksumSize); err != nil {
		return append_file_new, errors.As(err)
	}
	eq := bytes.Equal(aData, bData)
	if eq {
		if aSize == bSize {
			return append_file_complete, nil
		}
		return append_file_continue, nil
	}
	return append_file_new, nil
}

func travelFile(path string) (os.FileInfo, []string, error) {
	fStat, err := os.Lstat(path)
	if err != nil {
		return nil, nil, errors.As(err, path)
	}
	if !fStat.IsDir() {
		return nil, []string{path}, nil
	}
	dirs, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, nil, errors.As(err)
	}
	result := []string{}
	for _, fs := range dirs {
		filePath := filepath.Join(path, fs.Name())
		if !fs.IsDir() {
			result = append(result, filePath)
			continue
		}
		_, nextFiles, err := travelFile(filePath)
		if err != nil {
			return nil, nil, errors.As(err, filePath)
		}
		result = append(result, nextFiles...)
	}
	return fStat, result, nil
}

func copyFile(ctx context.Context, from, to string) error {
	if from == to {
		return errors.New("Same file").As(from, to)
	}
	if err := os.MkdirAll(filepath.Dir(to), 0755); err != nil {
		return errors.As(err)
	}
	fromFile, err := os.Open(from)
	if err != nil {
		return errors.As(err)
	}
	defer fromFile.Close()
	fromStat, err := fromFile.Stat()
	if err != nil {
		return errors.As(err)
	}

	// TODO: make chtime
	toFile, err := os.OpenFile(to, os.O_RDWR|os.O_CREATE, fromStat.Mode())
	if err != nil {
		return errors.As(err)
	}
	defer toFile.Close()
	toStat, err := toFile.Stat()
	if err != nil {
		return errors.As(err)
	}

	// checking continue
	stats, err := canAppendFile(fromFile, toFile, fromStat, toStat)
	if err != nil {
		return errors.As(err)
	}
	switch stats {
	case append_file_complete:
		// done
		fmt.Printf("%s ======= completed\n", to)
	case append_file_continue:
		appendPos := int64(toStat.Size() - 1)
		if appendPos < 0 {
			appendPos = 0
			fmt.Printf("%s ====== new \n", to)
		} else {
			fmt.Printf("%s ====== continue: %d\n", to, appendPos)
		}
		if _, err := fromFile.Seek(appendPos, 0); err != nil {
			return errors.As(err)
		}
		if _, err := toFile.Seek(appendPos, 0); err != nil {
			return errors.As(err)
		}
	default:
		fmt.Printf("%s ====== new \n", to)
		if _, err := fromFile.Seek(0, 0); err != nil {
			return errors.As(err)
		}
		if err := toFile.Truncate(0); err != nil {
			return errors.As(err)
		}
		if _, err := toFile.Seek(0, 0); err != nil {
			return errors.As(err)
		}
	}

	errBuff := make(chan error, 1)
	interrupt := false
	iLock := sync.Mutex{}
	go func() {
		for {
			iLock.Lock()
			if interrupt {
				iLock.Unlock()
				return
			}
			iLock.Unlock()

			if _, err := io.CopyN(toFile, fromFile, 32*1024); err != nil {
				errBuff <- errors.As(err)
				return
			}
		}
	}()
	select {
	case err := <-errBuff:
		if !errors.Equal(err, io.EOF) {
			return errors.As(err)
		}
		return nil
	case <-ctx.Done():
		iLock.Lock()
		interrupt = true
		iLock.Unlock()
		return ctx.Err()
	}
}

func CopyFile(ctx context.Context, from, to string) error {
	_, source, err := travelFile(from)
	if err != nil {
		return errors.As(err)
	}
	for _, src := range source {
		toFile := strings.Replace(src, from, to, 1)
		tCtx, cancel := context.WithTimeout(ctx, time.Hour)
		if err := copyFile(tCtx, src, toFile); err != nil {
			cancel()
			tCtx, cancel = context.WithTimeout(ctx, time.Hour)
			// do retry
			if err := copyFile(tCtx, src, toFile); err != nil {
				cancel()
				return errors.As(err)
			}
		}
		cancel()
	}
	return nil
}
