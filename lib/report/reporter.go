package report

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"

	"github.com/gwaylib/errors"
	"github.com/gwaylib/log"
)

type Reporter struct {
	lk        sync.Mutex
	serverUrl string

	reports chan []byte

	ctx     context.Context
	cancel  func()
	running bool
}

func NewReporter(buffer int) *Reporter {
	ctx, cancel := context.WithCancel(context.TODO())
	r := &Reporter{
		reports: make(chan []byte, buffer),

		ctx:    ctx,
		cancel: cancel,
	}

	go r.Run()
	return r
}

func (r *Reporter) send(data []byte) error {
	defer func() {
		if r := recover(); r != nil {
			log.Error(fmt.Sprintf("%+v,%s", r, string(debug.Stack())))
		}
	}()

	r.lk.Lock()
	defer r.lk.Unlock()
	if len(r.serverUrl) == 0 {
		return nil
	}
	resp, err := http.Post(r.serverUrl, "encoding/json", bytes.NewReader(data))
	if err != nil {
		return errors.As(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return errors.New("server response not success").As(resp.StatusCode, resp.Status)
	}
	return nil
}

func (r *Reporter) Run() {
	r.lk.Lock()
	if r.running {
		r.lk.Unlock()
		return
	}
	r.running = true
	r.lk.Unlock()

	errBuff := [][]byte{}
	for {
		select {
		case data := <-r.reports:
			if err := r.send(data); err != nil {
				log.Warn(errors.As(err))

				errBuff = append(errBuff, data)
				continue
			} else if len(errBuff) > 0 {
				sendIdx := 0
				for i, d := range errBuff {
					if err := r.send(d); err != nil {
						break
					}
					sendIdx = i
				}
				// send all success, clean buffer
				if sendIdx == len(errBuff)-1 {
					errBuff = [][]byte{}
				} else {
					errBuff = errBuff[sendIdx:]
				}
			}
		case <-r.ctx.Done():
		}
	}
}

func (r *Reporter) SetUrl(url string) {
	r.lk.Lock()
	defer r.lk.Unlock()
	r.serverUrl = url
}

func (r *Reporter) Send(data []byte) {
	r.reports <- data
}

func (r *Reporter) Close() error {
	r.lk.Lock()
	r.running = false
	r.lk.Unlock()

	r.cancel()
	return nil
}
