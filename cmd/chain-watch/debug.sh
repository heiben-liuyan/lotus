#!/bin/sh

#./chain-watch --repo=/data/sdb/lotus-user-1/.lotus run --kafka-cert="/root/hlm-miner/etc/kafka-cert" --kafka-user="hlmkafka" --kafka-pwd="HLMkafka2019" --kafka-addr="kf1.grandhelmsman.com:9093 kf2.grandhelmsman.com:9093 kf3.grandhelmsman.com:9093" --kafka-topic="browser_dev"
./chain-watch --repo=/data/sdb/lotus-user-1/.lotus run --kafka-cert="/root/hlm-miner/etc/kafka-cert" --kafka-user="hlmkafka" --kafka-pwd="HLMkafka2019" --kafka-addr="kf1.grandhelmsman.com:9093 kf2.grandhelmsman.com:9093 kf3.grandhelmsman.com:9093" --kafka-topic="browser_test"

# nohup ./debug.sh >>tmp.log 2>&1 &
# killall chain-watch
