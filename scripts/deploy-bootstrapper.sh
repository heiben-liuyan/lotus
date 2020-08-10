#!/usr/bin/env bash

log() {
  echo -e "\e[33m$1\e[39m"
}

host=$1

log "> Deploying bootstrap node $host"
log "Stopping lotus daemon"

ssh "$host" 'systemctl stop lotus-daemon' &
ssh "$host" 'systemctl stop lotus-miner' &

wait

ssh "$host" 'rm -rf .lotus' &
ssh "$host" 'rm -rf .lotusminer' &

scp -C lotus "${host}":/usr/local/bin/lotus &
scp -C lotus-miner "${host}":/usr/local/bin/lotus-miner &

wait

log 'Initializing repo'

ssh "$host" 'systemctl start lotus-daemon'
scp scripts/bootstrap.toml "${host}:.lotus/config.toml"
ssh "$host" "echo -e '[Metrics]\nNickname=\"Boot-$host\"' >> .lotus/config.toml"
ssh "$host" 'systemctl restart lotus-daemon'

sleep 10

log 'Extracting addr info'
ssh "$host" 'lotus net listen' | grep -v '/172' | grep -v '/ip6' > scripts/bootstrappers.pi
sed -i "s/127.0.0.1\/tcp\/1347/120.77.213.165\/tcp\/1346/g" scripts/bootstrappers.pi

log 'Connect to t0111'
ssh "$host" 'lotus net connect $(lotus --repo=/data/lotus/dev/.ldt0111 net listen)'

log 'Get fil from t0111'
ssh "$host" 'lotus wallet new bls'
ssh "$host" 'lotus --repo=/data/lotus/dev/.ldt0111 send $(lotus wallet default) 40000000'
git checkout build
