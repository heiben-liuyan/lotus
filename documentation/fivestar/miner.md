# 搭建开发环境

- [开发环境安装](#开发环境安装)
- [国内安装技巧](#国内安装技巧)
- [下载lotus源代码](#下载lotus源代码)
- [调试RUST](#调试RUST)
- [创建本地开发网络](#搭建创世节点)
    - [搭建存储节点](#搭建存储节点)
    - [接入本地开发网](#接入本地开发网)
- [目录规范](#目录规范)
    - [存储节点上的目录](#存储节点上的目录)
    - [链节点目录](#链节点目录)
    - [矿工节点目录](#矿工节点目录)
    - [计算节点目录](#计算节点目录)
- [发布二进制](#发布二进制)
- [附录](#附录)

## 开发环境安装
```shell
# 安装依赖(需要ubuntu 18.04)
apt-get update
apt-get install aptitude
aptitude install chrony nfs-common gcc git bzr jq pkg-config mesa-opencl-icd ocl-icd-opencl-dev 
```

## 国内安装技巧 
参考: https://docs.lotu.sh/en+install-lotus-ubuntu

### 1), 安装go
```shell
sudo su -
cd /usr/local/
wget https://studygolang.com/dl/golang/go1.14.4.linux-amd64.tar.gz # 其他版本请参考https://studygolang.com/dl
tar -xzf go1.14.4.linux-amd64.tar.gz
### 配置/etc/profile环境变量(需要重新登录生效或source /etc/profile)
export GOROOT=/usr/local/go
export GOPROXY="https://goproxy.io,direct"
export GOPRIVATE="github.com/filecoin-fivestar"
export GIT_TERMINAL_PROMPT=1
export PATH=$GOROOT/bin:$PATH:/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games:/snap/bin
exit # 退出sudo su -
```

### 2)，安装rust
```shell
mkdir ~/.cargo

### 设置国内镜像代理(或设置到~/.profile中, 需要重新登录生效或source ~/.profile))
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

## 下载lotus源代码
```shell
mkdir -p $HOME/go/src/github.com/filecoin-project
cd $HOME/go/src/github.com/filecoin-project
git clone https://github.com/filecoin-fivestar/lotus.git lotus
cd lotus
```

## 调试RUST
```shell
mkdir -p $HOME/go/src/github.com/filecoin-project
cd $HOME/go/src/github.com/filecoin-project
git clone https://github.com/filecoin-fivestar/lotus.git lotus
git clone https://github.com/filecoin-project/rust-fil-proofs.git
git clone https://https://github.com/filecoin-project/rust-filecoin-proofs-api.git
```
### 在rust-fil-proofs下测试
``` 
cd $HOME/go/src/github.com/filecoin-project/rust-fil-proofs
RUST_BACKTRACE=1 RUST_LOG=info FIL_PROOFS_USE_GPU_TREE_BUILDER=1 FIL_PROOFS_USE_GPU_COLUMN_BUILDER=1 cargo run --release --bin benchy -- stacked --size 2
```
### 在lotus下测试
1, 修改lotus/extern/filecoin-ffi/rust/Cargo.toml指向
```
[dependencies.filecoin-proofs-api]
package = "filecoin-proofs-api"
#version = "4.0.2"
path = "../../../../rust-filecoin-proofs-api"
```

2, 切换rust-filecoin-proofs-api版本与指向
```shell
cd $HOME/go/src/github.com/filecoin-project/rust-filecoin-proofs-api
git checkout v4.0.2 # 需要与lotus使用的同一版本
```

修改rust-filecoin-proofs-api/Cargo.toml指向
```
[dependencies]
anyhow = "1.0.26"
serde = "1.0.104"
paired = "0.20.0"
#filecoin-proofs-v1 = { package = "filecoin-proofs", version = "4.0.2" }
filecoin-proofs-v1 = { package = "filecoin-proofs", path = "../rust-fil-proofs/filecoin-proofs" }
```

3, 切换rust-fil-proofs版本与指向
```shell
cd $HOME/go/src/github.com/filecoin-project/rust-filecoin-proofs-api
git checkout releases/v4.0.2 # 需要与rust-filecoin-proofs-api使用的同一版本
```

4, 编译lotus基测程序
```shell
cd $HOME/go/src/github.com/filecoin-project/lotus
make clean
env RUSTFLAGS="-C target-cpu=native -g" FFI_BUILD_FROM_SOURCE=1 make bench
./bensh.sh
```

## 搭建创世节点
```shell
./clean-bootstrap.sh
ps axu|grep lotus # 确认所有相关进程已关闭
./init-bootstrap.sh
tail -f boostrap.log # 直到Heaviest tipset 有10来个高度左右, ctrl+c 退出
ssh-keygen -t ed25519 # 创建本机ssh密钥信息，已有跳过
./deploy-boostrap.sh # 部署水龙头及对外提供的初始节点
```

## 创建本地开发网络

### 搭建存储节点
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

### 接入本地开发网
```shell
./install.sh debug # 若是使用正式，执行./install.sh进行编译, 编译完成后自动放在$FILECOIN_BIN下
rm -rf /data/sdb/lotus-user-1/.lotus* # 注意!!!! 需要确认此库不是正式库，删掉需要重新同步数据与创建矿工，若创世节点一样，可不删除。
```

shell 1, 运行链
```
cd ../../scripts/fivestar
./daemon.sh
```

shell 2, 创建私网矿工, 首次运行时需要构建, 或通过浏览器来创建
```
cd ../../scripts/fivestar
./init-miner-dev.sh
```

shell 3, 运行矿工
```
cd ../../scripts/fivestar
./miner.sh
```

shell 4, 运行worker
```
cd ../../scripts/fivestar
./worker.sh
```

shell 5，操作miner
```
cd ../../scripts/fivestar

# 添加存储节点
./init-storage-dev.sh

# 运行刷量
./mshell.sh pledge-sector start

# miner的其他指令，参阅
./mshell.sh --help
```

## 目录规范

项目涉及到的所有目录，以下这些目录将在单机部署上建立
```
/data -- 项目数据目录

# 缓存盘
/data/cache -- 缓存盘，必要时此盘数据会被清除，存放的数据要求是可损坏的，可单独挂载盘，建议挂载ssd盘
/data/cache/filecoin-proof-parameters -- filecoin本地启动参数版本管理目录文件，此文件数据需要65G左右的空间
/data/cache/filecoin-proof-parameters/v20 -- filecoin本地启动参数目录实际目文件
/data/cache/.lotusworker -- lotus-seal-worker计算缓存目录，计算结束后会自动清除，需要1T左右空间
/data/cache/.lotusworker/push -- 计算结果推送目录，会自动单独挂载盘，可选
/data/cache/tmp -- 程序$TMPDIR设定的目录

# lotus公共参数数据，可单独挂载盘
/data/lotus
/data/lotus/filecoin-proof-parameters -- lotus启动参数文件，可单独挂载盘; 可选，用于提供parameters的下载
/data/lotus/filecoin-proof-parameters/v20 -- lotus对应版本的启动参数，若存在，worker脚本会同步复到到/data/cache/filecoin-proof-parameters下

# 矿工数据盘
/data/sd(?) -- 矿工存储数据目录(前期设计多进程时对应多盘位), 可单独挂载盘，默认为/data/sdb
/data/sd(?)/lotus-user-1/.lotus -- lotus矿工绑定的数据链目录, 可单独挂载盘, 默认为/data/sdb/lotus-user-1/.lotus
/data/sd(?)/lotus-user-1/.lotusstorage -- lotus矿工存储数据目录, 可单独挂载盘, 默认为/data/sdb/lotus-user-1/.lotusstorage


# 存储链接入口
/data/zfs -- 挂载zfs池到本地的目录
/data/nfs -- 挂载nfs文件的目录

# 启动参数链接入口
/var/tmp/filecoin-proof-parameters # filecoin启动参数文件入口，会被软连接到/data/cache/filecoin-proof-parameters对应版本下
```


### 存储节点上的目录

```
/data/zfs的目录自行挂载需要的盘

```

配置nfs的/etc/exports文件,进行nfs导出
```
/data/zfs/ *(rw,sync,insecure,no_root_squash)
```

### 链节点目录
链同步节点目录, 用于存储区块链数据，长期考虑，应留1T的链数据空间或挂载为长期的存储盘，应与矿工数据分离存储
```text
/data/sd(?)/lotus-user-x/.lotus # 默认为/data/sdb/lotus-user-1/.lotus
```
### 矿工节点目录

在miner节点中，会用到三种级别的目录

链api目录, 若是同一台机器，不需要新建
```text
/data/sd(?)/lotus-user-x/.lotus # 默认为/data/sdb/lotus-user-1/.lotus
```

矿工元数据节点目录, 用于引导miner的启动
```text
/data/sd(?)/lotus-user-x/.lotusstorage # 默认为/data/sdb/lotus-user-1/.lotus
```

存力存储目录, 用于实际存储存力, 由矿工节点自动进行管理与挂载
```text
/data/nfs/1
/data/nfs/2
/data/nfs/3
```

### 计算节点目录

在worker中，需要用到三个目录

矿工api配置文件目录，用于启动worker, 若在同一台机器上，不需要新建
```text
/data/sdx/lotus-user-x/.lotusstorage
```

工作者配置文件目录，用于缓存存储临时密封的数据, 应使用高速io盘，以便提高本地的io吞吐
```text
/data/cache/.lotusworker
```

密封结果推送目录
```text
/data/cache/.lotusworker/push

worker程序会根据miner分发的存储节点配置自动挂载
```

## 发布二进制
将发布到./deploy/lotus
```
./publish.sh linux-amd64-amd
```

## 附录

[开发IDE](https://github.com/filecoin-fivestar/ide)
