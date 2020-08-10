#!/bin/sh

export IPFS_GATEWAY="https://proof-parameters.s3.cn-south-1.jdcloud-oss.com/ipfs/"

repodir=$1
if [ -z "$repodir" ]; then
    repodir=/data/sdb/lotus-user-1/.lotus
fi
miner_repodir=$2
if [ -z "$miner_repodir" ]; then
    miner_repodir=/data/sdb/lotus-user-1/.lotusstorage
fi

mkdir -p $repodir
mkdir -p $miner_repodir

RUST_LOG=info RUST_BACKTRACE=1 ../../lotus-miner --repo=$repodir --miner-repo=$miner_repodir run --nosync=true &
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

