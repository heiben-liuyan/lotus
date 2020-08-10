package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	cluster "github.com/bsm/sarama-cluster" //support automatic consumer-group rebalancing and offset tracking

	//"github.com/sdbaiguanghe/glog"
	"github.com/gwaylib/errors"
)

var (
	_kp     sarama.SyncProducer
	_kpLock = sync.Mutex{}
)

func closeKafkaProducer() {
	if _kp != nil {
		_kp.Close()
	}
	time.Sleep(3e9)
}

func getKafkaProducer() (sarama.SyncProducer, error) {
	if _kp != nil {
		return _kp, nil
	}
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Timeout = 5 * time.Second
	config.Net.SASL.Enable = true
	config.Net.SASL.Handshake = true
	config.Net.SASL.User = _kafkaUser
	config.Net.SASL.Password = _kafkaPasswd
	//证书位置
	kafkaCert := _kafkaCertFile
	certBytes, err := ioutil.ReadFile(kafkaCert)
	if err != nil {
		return nil, errors.As(err, kafkaCert)
	}
	clientCertPool := x509.NewCertPool()
	ok := clientCertPool.AppendCertsFromPEM(certBytes)
	if !ok {
		return nil, errors.New("kafka producer failed to parse root certificate").As(kafkaCert)
	}
	config.Net.TLS.Config = &tls.Config{
		//Certificates:       []tls.Certificate{},
		RootCAs:            clientCertPool,
		InsecureSkipVerify: true,
	}

	config.Net.TLS.Enable = true
	address := _kafkaAddress
	p, err := sarama.NewSyncProducer(address, config)
	if err != nil {
		return nil, errors.As(err, address)
	}
	_kp = p
	return _kp, nil
}

//生产消息模式
func KafkaProducer(producerData string, topic string) error {
	// TODO: fix this to pool
	go func(m string) {
		_kpLock.Lock()
		defer _kpLock.Unlock()
		p, err := getKafkaProducer()
		if err != nil {
			log.Error(err)
			closeKafkaProducer()
			return
		}
		msg := &sarama.ProducerMessage{
			Topic: topic,
			Value: sarama.ByteEncoder(producerData),
		}
		log.Infof("send kafka msg:%s", m)
		part, offset, err := p.SendMessage(msg)
		if err != nil {
			log.Error(err)
			closeKafkaProducer()
			return
		}
		log.Infof("send kafka msg done:%s, partition:%d,offset:%d", m, part, offset)
	}(producerData)

	return nil
}

func KafkaConsumer(groupID string, topics []string) []byte {
	config := cluster.NewConfig()
	config.Consumer.Return.Errors = true
	config.Group.Return.Notifications = true
	config.Net.SASL.Enable = true
	config.Net.SASL.User = _kafkaUser
	config.Net.SASL.Password = _kafkaPasswd
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	//证书位置
	kafkaCert := _kafkaCertFile
	certBytes, err := ioutil.ReadFile(kafkaCert)
	clientCertPool := x509.NewCertPool()
	ok := clientCertPool.AppendCertsFromPEM(certBytes)
	if !ok {
		panic("kafka producer failed to parse root certificate")
	}

	config.Net.TLS.Config = &tls.Config{
		//Certificates:       []tls.Certificate{},
		RootCAs:            clientCertPool,
		InsecureSkipVerify: true,
	}

	config.Net.TLS.Enable = true
	address := _kafkaAddress
	// init consumer
	consumer, err := cluster.NewConsumer(address, groupID, topics, config)
	if err != nil {
		panic(err)
	}
	defer consumer.Close()
	// trap SIGINT to trigger a shutdown.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	// consume errors
	go func() {
		for err := range consumer.Errors() {
			log.Warnf("Error: %s\n", err.Error())
		}
	}()
	// consume notifications
	go func() {
		for ntf := range consumer.Notifications() {
			log.Warnf("Rebalanced: %+v\n", ntf)
		}
	}()
	// consume messages, watch signals
	select {
	case msg, ok := <-consumer.Messages():
		if ok {
			fmt.Fprintf(os.Stdout, "%s/%d/%d\t%s\t%s\n", msg.Topic, msg.Partition, msg.Offset, msg.Key, msg.Value)
			consumer.MarkOffset(msg, "") // mark message as processed
			return msg.Value
		}
	case <-signals:
		return nil
	}
	return nil
}
