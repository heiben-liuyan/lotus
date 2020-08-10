#!/bin/sh

# 1GB: 1073741824
# 32GB: 34359738368
# 100GB: 107374182400
# 1TB: 1099511627776
# 8TB: 8796093022208
# 15TB: 16492674416640
# 1PB: 1125899906842624

# for local, 1TB limit, when need to scale, see ./miner.sh hlm-storage scale --help
netip=$(ip a | grep -Po '(?<=inet ).*(?=\/)'|grep -E "10\.") # only support one eth card.
./mshell.sh hlm-storage add --mount-type="nfs" --mount-signal-uri="$netip:/data/zfs" --mount-dir="/data/nfs" --max-size=1099511627776 --max-work=100

## for testing in machine room
## set hlm-storage with 1TB for testing scale.
#./miner.sh hlm-storage add --mount-type=nfs --mount-dir=/data/nfs --max-size=-1 --keep-size=1099511627776 --max-work=100 --mount-signal-uri=10.1.30.2:/data/zfs
#./miner.sh hlm-storage add --mount-type=nfs --mount-dir=/data/nfs --max-size=-1 --keep-size=1099511627776 --max-work=100 --mount-signal-uri=10.1.30.3:/data/zfs
#./miner.sh hlm-storage add --mount-type=nfs --mount-dir=/data/nfs --max-size=-1 --keep-size=1099511627776 --max-work=100 --mount-signal-uri=10.1.30.4:/data/zfs

