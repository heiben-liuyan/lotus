#!/bin/sh
export IPFS_GATEWAY="https://proof-parameters.s3.cn-south-1.jdcloud-oss.com/ipfs/"
# Note that FIL_PROOFS_USE_GPU_TREE_BUILDER=1 is for tree_r_last building and FIL_PROOFS_USE_GPU_COLUMN_BUILDER=1 is for tree_c.  
# So be sure to use both if you want both built on the GPU
export FIL_PROOFS_USE_GPU_COLUMN_BUILDER=0
export FIL_PROOFS_USE_GPU_TREE_BUILDER=0
export FIL_PROOFS_MAXIMIZE_CACHING=1  # open cache for 32GB or 64GB

# checking gpu
gpu=""
type nvidia-smi
if [ $? -eq 0 ]; then
    gpu=$(nvidia-smi -L|grep "GeForce")
fi
if [ ! -z "$gpu" ]; then
    FIL_PROOFS_USE_GPU_COLUMN_BUILDER=1
    FIL_PROOFS_USE_GPU_TREE_BUILDER=1
fi


miner_repo=$1 # miner_repo of go-filecoin


if [ -z "$miner_repo" ]; then
    miner_repo=/data/sdb/lotus-user-1/.lotusstorage
fi

tmp=/data/cache/tmp
repo=$2
if [ -z "$repo" ]; then
    repo=/data/cache/.lotusworker 
fi
storage_repo=$3
if [ -z "$storage_repo" ]; then
    storage_repo=$repo/push # 密封结果存放
fi

mkdir -p $tmp
mkdir -p $repo
mkdir -p $miner_repo
mkdir -p $storage_repo

# check parameters
if [ ! -d /var/tmp/filecoin-proof-parameters ]; then
    pVer="v27"
    if [ ! -d /data/cache/filecoin-proof-parameters/$pVer ]; then
    	mkdir -p /data/cache/filecoin-proof-parameters/$pVer
    fi

    # clean cache, and only keep the $pVer one
    for name in `ls /data/cache/filecoin-proof-parameters`
    do
        if [ $name = $pVer ]; then
            continue
        fi
        rm -rf /data/cache/filecoin-proof-parameters/$name
    done

    # download parameters if the source exist.
    if [ -d /data/lotus/filecoin-proof-parameters/$pVer ]; then
        rsync -Pat /data/lotus/filecoin-proof-parameters/$pVer/ /data/cache/filecoin-proof-parameters/$pVer/
        if [ $? -ne 0 ]; then
            echo "rsync failed"
            sleep 3
            exit 0
        else
            # release connection
            umount -fl /data/lotus/filecoin-proof-parameters
        fi
    fi

    # link
    ls /var/tmp/filecoin-proof-parameters 2>&1 >/dev/nul
    if [ $? -ne 0 ]; then  
        rm -rf /var/tmp/filecoin-proof-parameters
        ln -s /data/cache/filecoin-proof-parameters/$pVer /var/tmp/filecoin-proof-parameters
    fi
else
    echo "/var/tmp/filecoin-proof-parameters already exist."
fi

netip=$(ip a | grep -Po '(?<=inet ).*(?=\/)'|grep -E "10\.|172\.|192\.") # only support one eth card.
RUST_LOG=info RUST_BACKTRACE=1 NETIP=$netip ./lotus-worker --repo=$repo --miner-repo=$miner_repo --storage-repo=$storage_repo run &
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

if [ ! -z "$pid" ]; then
    wait "$pid"
fi
