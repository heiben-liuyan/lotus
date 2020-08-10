#!/bin/sh

#host="http://120.77.213.165:7776"
host="http://127.0.0.1:7777"
size=2048
#size=536870912 # 512MB
#size=34359738368 # 32G

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

walletAddr=$(../../lotus --repo=$repodir wallet new bls)
echo $walletAddr
mkresp=$(curl -s $host"/mkminer?sectorSize=$size&address="$walletAddr)

echo $mkresp
f=$(echo $mkresp|awk -F 'a href="/wait.html\\?f=' '{print $2}'|awk -F '&amp;m=' '{print $1}')
if [ -z "$f" ]; then
    echo "f not found"
    exit
fi

curl -s $host"/msgwait?cid="$f
minerAddrJs=$(curl -s $host"/msgwaitaddr?cid="$f)
minerAddr=$(echo $minerAddrJs|awk -F '"addr": "' '{print $2}'|awk -F '"}' '{print $1}')

echo lotus-miner init --actor=$minerAddr --owner=$walletAddr
../../lotus-miner --repo=$repodir --miner-repo=$miner_repodir init --actor=$minerAddr --owner=$walletAddr

cp ./config-miner.toml $miner_repodir/config.toml
netip=$(ip a | grep -Po '(?<=inet ).*(?=\/)'|grep -E "10\.") # only support one eth card.
echo "Set $netip to config.toml"
sed -i "s/127.0.0.1/$netip/g" $miner_repodir/config.toml

