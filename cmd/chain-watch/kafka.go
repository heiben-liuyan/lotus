package main

import (
	"time"

	"github.com/google/uuid"
)

var (
	_kafkaAddress = []string{}
	_kafkaTopic   = ""

	_kafkaUser     = ""
	_kafkaPasswd   = ""
	_kafkaCertFile = ""
)

// 协议共公部分

type KafkaCommon struct {
	KafkaId        string // 协议唯一ID
	KafkaTimestamp int64  // 发送时间
	Type           string // 协议类型
}

func GenKID() string {
	return uuid.New().String()
}
func GenKTimestamp() int64 {
	return time.Now().UnixNano()
}
