# Installation instructions

## Development environment installation
```shell
# depends on (need ubuntu 18.04)
apt-get update
apt-get install aptitude
aptitude install chrony nfs-common gcc git bzr jq pkg-config mesa-opencl-icd ocl-icd-opencl-dev 
```

## Installation skills in China
Refer to: https://docs.lotu.sh/en+install-lotus-ubuntu

### 1), Install go
```shell
sudo su -
cd /usr/local/
wget https://studygolang.com/dl/golang/go1.14.4.linux-amd64.tar.gz
tar -xzf go1.14.4.linux-amd64.tar.gz
### edit /etc/profile(relogin or source /etc/profile for apply)
export GOROOT=/usr/local/go
export GOPROXY="https://goproxy.io,direct"
export GOPRIVATE="github.com/filecoin-fivestar"
export GIT_TERMINAL_PROMPT=1
export PATH=$GOROOT/bin:$PATH:/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games:/snap/bin
exit # 退出sudo su -
```

### 2)，Install rust
```shell
mkdir ~/.cargo

export RUSTUP_DIST_SERVER=https://mirrors.ustc.edu.cn/rust-static
export RUSTUP_UPDATE_ROOT=https://mirrors.ustc.edu.cn/rust-static/rustup

cat > ~/.cargo/config <<EOF
[source.crates-io]
registry = "https://github.com/rust-lang/crates.io-index"
# 指定镜像
replace-with = 'sjtu'

# 清华大学
[source.tuna]
registry = "https://mirrors.tuna.tsinghua.edu.cn/git/crates.io-index.git"

# 中国科学技术大学
[source.ustc]
registry = "git://mirrors.ustc.edu.cn/crates.io-index"

# 上海交通大学
[source.sjtu]
registry = "https://mirrors.sjtug.sjtu.edu.cn/git/crates.io-index"

# rustcc社区
[source.rustcc]
registry = "https://code.aliyun.com/rustcc/crates.io-index.git"
EOF

curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
```

## Install NFS in local net
```shell
aptitude install nfs-server
mkdir -p /data/nfs
mkdir -p /data/zfs
mkdir -p /data/zfs/cache
mkdir -p /data/zfs/sealed
chattr -V +a /data/zfs
chattr -V +a /data/zfs/cache
chattr -V +a /data/zfs/sealed

echo "/data/zfs/ *(rw,sync,insecure,no_root_squash)" >>/etc/exports
systemctl reload nfs-server
```

## Download lotus code
```shell
mkdir -p $HOME/go/src/github.com/filecoin-project
cd $HOME/go/src/github.com/filecoin-project
git clone https://github.com/filecoin-fivestar/lotus.git lotus
cd lotus
```

## RUST Develop(Only for debug the rust code)
```shell
mkdir -p $HOME/go/src/github.com/filecoin-project
cd $HOME/go/src/github.com/filecoin-project
git clone https://github.com/filecoin-fivestar/lotus.git lotus
git clone https://github.com/filecoin-project/rust-fil-proofs.git
git clone https://https://github.com/filecoin-project/rust-filecoin-proofs-api.git
```
### Testing in rust-fil-proofs
``` 
cd $HOME/go/src/github.com/filecoin-project/rust-fil-proofs
RUST_BACKTRACE=1 RUST_LOG=info FIL_PROOFS_USE_GPU_TREE_BUILDER=1 FIL_PROOFS_USE_GPU_COLUMN_BUILDER=1 cargo run --release --bin benchy -- stacked --size 2
```
### Testing in lotus
1, Changing the `lotus/extern/filecoin-ffi/rust/Cargo.toml`
```
[dependencies.filecoin-proofs-api]
package = "filecoin-proofs-api"
#version = "4.0.2"
path = "../../../rust-filecoin-proofs-api"
```

2, Changing the `rust-filecoin-proofs-api`
```shell
cd $HOME/go/src/github.com/filecoin-project/rust-filecoin-proofs-api
git checkout v4.0.2 
```

Edit `rust-filecoin-proofs-api/Cargo.toml`
```
[dependencies]
anyhow = "1.0.26"
serde = "1.0.104"
paired = "0.20.0"
#filecoin-proofs-v1 = { package = "filecoin-proofs", version = "4.0.2" }
filecoin-proofs-v1 = { package = "filecoin-proofs", path = "../rust-fil-proofs/filecoin-proofs" }
```

3, Changing the `rust-fil-proofs`
```shell
cd $HOME/go/src/github.com/filecoin-project/rust-filecoin-proofs-api
git checkout releases/v4.0.2 # Same as which the proofs-api using.
```

4, Building the lotus-bench
```shell
cd $HOME/go/src/github.com/filecoin-project/lotus
make clean
env RUSTFLAGS="-C target-cpu=native -g" FFI_BUILD_FROM_SOURCE=1 make bench
./bensh.sh
```

## Building the genesis node of local devnet
```shell
./clean-bootstrap.sh
ps axu|grep lotus # Confirm all the lotus process has shutdown.
./init-bootstrap.sh
tail -f boostrap.log # Waiting for Heaviest tipset height has about 10, exit by ctrl+c.
ssh-keygen -t ed25519 # if done, skip this step.
./deploy-boostrap.sh 
```

## Connect ot the genesis node of local devnet
```shell
./install.sh debug # if $FILECOIN_BIN has set, install to $FILECOIN_BIN.
rm -rf /data/sdb/lotus-user-1/.lotus* # SPEC: please confirm the data is not important!!!
```

shell 1, running lotus-daemon
```
cd ../../scripts/fivestar
./daemon.sh
```

shell 2, create a miner.
```
cd ../../scripts/fivestar
./init-miner-dev.sh
```

shell 3, running the miner.
```
cd ../../scripts/fivestar
./miner.sh
```

shell 4, running a worker
```
cd ../../scripts/fivestar
./worker.sh
```

shell 5, operating on miner
```
cd ../../scripts/fivestar

# Import a storage node(with local nfs)
./init-storage-dev.sh

# Running the pledge-sector
./mshell.sh pledge-sector start

# other miner more for:
./mshell.sh --help
```

## Directory specs(目录规范)

The following directories will be created on a stand-alone deployment
```
/data -- All data directory of the project

# SSD cache of seal worker
/data/cache -- 缓存盘，必要时此盘数据会被清除，存放的数据要求是可损坏的，可单独挂载盘，建议挂载1T ssd盘
/data/cache/filecoin-proof-parameters -- filecoin本地启动参数版本管理目录文件，此文件数据需要65G左右的空间
/data/cache/filecoin-proof-parameters/v20 -- filecoin本地启动参数目录实际目文件
/data/cache/.lotusworker -- lotus-seal-worker计算缓存目录，计算结束后会自动清除，需要1T左右空间
/data/cache/.lotusworker/push -- 计算结果推送目录，会自动单独挂载盘，可选
/data/cache/tmp -- 程序$TMPDIR设定的目录

# Data source of lotus (lotus数据源)
/data/lotus
/data/lotus/filecoin-proof-parameters -- lotus启动参数文件，可单独挂载盘; 可选，用于提供parameters的下载
/data/lotus/filecoin-proof-parameters/v20 -- lotus对应版本的启动参数，若存在，worker脚本会同步复到到/data/cache/filecoin-proof-parameters下

# Data disk for miner (矿工数据盘)
/data/sd(?) -- 矿工存储数据目录(前期设计多进程时对应多盘位), 可单独挂载盘，默认为/data/sdb
/data/sd(?)/lotus-user-1/.lotus -- lotus矿工绑定的数据链目录, 可单独挂载盘, 默认为/data/sdb/lotus-user-1/.lotus
/data/sd(?)/lotus-user-1/.lotusstorage -- lotus矿工存储数据目录, 可单独挂载盘, 默认为/data/sdb/lotus-user-1/.lotusstorage


# Storage node interface(存储节点链接入口)
/data/zfs -- Mount local disk to support a storage node. (挂载zfs池到本地的目录)
/data/nfs -- Mount the remote nfs contact to miner node. (挂载nfs文件的目录到矿节点)

# filecoin-proof-parameters directory for boot the program. (启动参数链接入口)
/var/tmp/filecoin-proof-parameters # link to /data/cache/filecoin-proof-parameters/$ver
```


### Directory of storage node (存储节点目录)

```
/data/zfs -- Mount a disk or a mountable storage system to here.
```

Setup the nfs config file(/etc/exports) to export the /data/zfs
```
/data/zfs/ *(rw,sync,insecure,no_root_squash)
```

### Directory of miner node (矿工节点目录)

Directory for chain data
```text
/data/sd(?)/lotus-user-x/.lotus # Default is `/data/sdb/lotus-user-1/.lotus`
```

Directory for miner data
```text
/data/sd(?)/lotus-user-x/.lotusstorage # 默认为/data/sdb/lotus-user-1/.lotus
```

Directory for storage node, auto mount by lotus-miner.
```text
/data/nfs/1
/data/nfs/2
/data/nfs/3
```

### Directory of worker node. (计算节点工作目录)

Directory for miner auth api.
```text
/data/sdx/lotus-user-x/.lotusstorage
```

Directory for cache the seal data of lotus-worker.
```text
/data/cache/.lotusworker
```

Directory for push the sealed data to storage node. Auto mount storage node(nfs) to here by lotus-worker.
```text
/data/cache/.lotusworker/push
```


## Release the bin
publish data to ./deploy/lotus
```
./publish.sh linux-amd64-amd
```

