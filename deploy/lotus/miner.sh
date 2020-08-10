#!/bin/sh

export IPFS_GATEWAY="https://proof-parameters.s3.cn-south-1.jdcloud-oss.com/ipfs/"

repodir=$1
if [ -z "$repodir" ]; then
    repodir=/data/sdb/lotus-user-1/.lotus
fi
storagerepodir=$2
if [ -z "$storagerepodir" ]; then
    storagerepodir=/data/sdb/lotus-user-1/.lotusstorage
fi

mkdir -p $repodir
mkdir -p $storagerepodir

# BELLMAN_NO_GPU=1 RUST_LOG=info RUST_BACKTRACE=1 ./lotus-miner --repo=$repodir --storagerepo=$storagerepodir run --nosync 
#pid=$!
RUST_LOG=info RUST_BACKTRACE=1 ./lotus-miner --repo=$repodir --storagerepo=$storagerepodir run --nosync=true &
pid=$!
#./lotus-miner --repo=$repodir --storagerepo=$storagerepodir run --nosync=true &
#pid=$!


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

