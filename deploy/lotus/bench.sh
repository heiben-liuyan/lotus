#!/bin/sh

# REAME
# make bench
# nohup ./bench.sh &
# tail -f nohup.out
# REAME end

export IPFS_GATEWAY="https://proof-parameters.s3.cn-south-1.jdcloud-oss.com/ipfs/"

# Note that FIL_PROOFS_USE_GPU_TREE_BUILDER=1 is for tree_r_last building and FIL_PROOFS_USE_GPU_COLUMN_BUILDER=1 is for tree_c.  
# So be sure to use both if you want both built on the GPU
export FIL_PROOFS_USE_GPU_COLUMN_BUILDER=1
export FIL_PROOFS_USE_GPU_TREE_BUILDER=1 
export FIL_PROOFS_MAXIMIZE_CACHING=0  # open cache for 32GB or 64GB

#size=34359738368 # 32GB
#size=536870912 # 512MB
size=2048
RUST_LOG=info RUST_BACKTRACE=1 ./lotus-bench sealing --storage-dir=/data/cache/.lotus-bench --sector-size=$size #--parallel=1
#RUST_LOG=info RUST_BACKTRACE=1 ./bench sealing --storage-dir=/data/cache/.lotus-bench --sector-size=$size --parallel=2 --num-sectors=2

