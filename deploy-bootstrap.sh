#!/bin/sh


# nohup ./scripts/init-network.sh >>boostrap.log 2>&1 & # do this at first

./scripts/setup-host.sh root@127.0.0.1
./scripts/deploy-node.sh root@127.0.0.1
./scripts/deploy-bootstrapper.sh root@127.0.0.1
sleep 10
nohup ./lotus-fountain run --front=0.0.0.0:7777 --from=$(lotus wallet default) --amount=1280000 >lotus-fountain.log 2>&1 &
