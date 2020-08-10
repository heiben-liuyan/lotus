#!/bin/sh
systemctl stop lotus-daemon
killall lotus
killall lotus-miner
killall lotus-fountain
killall lotus-seed
rm -rf /data/lotus/dev
rm -rf ~/.lotus
ps axu|grep lotus
