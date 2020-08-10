package main

import (
	_ "github.com/lib/pq"
)

type storage struct {
}

func (s *storage) Write(data []byte) (int, error) {
	// TODO: send to kafka
	return 0, nil
}
