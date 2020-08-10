#!/bin/sh

export FILECOIN_BIN=./deploy/lotus

ver=""
if [ ! -z "$1" ]; then
    ver=$1"-"
fi
ver=$ver$(git describe --always --match=NeVeRmAtCh --dirty)

rm -rf ./deploy
git checkout deploy

./install.sh

if [ $? -ne 0 ]; then
    exit 1
fi

cd ./deploy
tar -czf lotus-$ver.tar.gz lotus

