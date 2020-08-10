#!/bin/sh

repodir=$1
if [ -z "$repodir" ]; then
    repodir=/data/sdb/lotus-user-1/.lotus
fi
mkdir -p $repodir

netip=$(ip a | grep -Po '(?<=inet ).*(?=\/)'|grep -E "10\.|172\.|192\.") # only support one eth card.
echo "Set $netip to config.toml"
cp config-lotus.toml $repodir/config.toml
sed -i "s/127.0.0.1/$netip/g" $repodir/config.toml

export IPFS_GATEWAY="https://proof-parameters.s3.cn-south-1.jdcloud-oss.com/ipfs/"

./lotus --repo=$repodir daemon &
pid=$!

# set ulimit for process
nropen=$(cat /proc/sys/fs/nr_open)
echo "max nofile limit:"$nropen
echo "current nofile of $pid limit:"$(cat /proc/$pid/limits|grep "open files")
prlimit -p $pid --nofile=$nropen
if [ $? -eq 0 ]; then
    echo "new nofile of $pid limit:"$(cat /proc/$pid/limits|grep "open files")
else
    echo "set prlimit failed, command:prlimit -p $pid --nofile=$nropen"
    exit 0
fi

wait "$pid"
