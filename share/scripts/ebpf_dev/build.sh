#!/usr/bin/env bash

#这个脚本只适合centos7, 如果是其他发行版, 请自行修改
#不好评估是否需要直接退出
#set -e

ROOT_DIR="/root"

cd "$ROOT_DIR" || exit 1


# 检查总内存, 要求不能小于4G

TOTAL_MEM=$(free -m | awk '/Mem/ {print $2}')
if [ "$TOTAL_MEM" -lt 4096 ]; then
    echo "内存小于4G, 请扩容docker后重试"
    exit 1
fi

# 检查是否已经安装bcc 如果 存在 /usr/share/bcc 目录, 则认为已经安装
if [ -d "/usr/share/bcc" ]; then
    echo "可能你的bcc已经安装, 请删除/usr/share/bcc目录后重试?"
    exit 1
fi



# 安装编译环境

yum install -y epel-release
yum update -y
yum groupinstall -y "Development tools"
yum install -y elfutils-libelf-devel cmake3 git bison flex ncurses-devel
yum install -y luajit luajit-devel
yum install -y vim


# 安装 clang/llvm

yum install -y centos-release-scl
yum-config-manager --enable rhel-server-rhscl-7-rpms

bash -c 'cat << EOF > /etc/yum.repos.d/llvmtoolset-build.repo
[llvmtoolset-build]
name=LLVM Toolset 10.0 - Build
baseurl=https://buildlogs.centos.org/c7-llvm-toolset-10.0.x86_64/
gpgcheck=0
enabled=1
EOF'

yum install -y devtoolset-9 \
    llvm-toolset-10.0 \
    llvm-toolset-10.0-llvm-devel \
    llvm-toolset-10.0-llvm-static \
    llvm-toolset-10.0-clang-devel || exit 1

source scl_source enable devtoolset-9 llvm-toolset-10.0

# 准备相关的环境变量

cp "${ROOT_DIR}/ebpf-enable.sh" /etc/profile.d/ || exit 1
echo 'export LLVM_ROOT=/opt/rh/llvm-toolset-10.0/root/usr/lib64/cmake/llvm' >> "${ROOT_DIR}/.bashrc"
source "${ROOT_DIR}/.bashrc"

# 编译安装bcc

cd "${ROOT_DIR}" || exit
git clone https://github.com/iovisor/bcc.git "${ROOT_DIR}/bcc" || exit 1
cd "${ROOT_DIR}/bcc" || exit 1
git apply "${ROOT_DIR}/cmake.patch" || exit 1
mkdir build; cd build || exit 1
cmake3 ..

# 获取cpu核心数, 执行make
CPU_CORE=$(grep -c ^processor /proc/cpuinfo)
make -j "$CPU_CORE" || exit 1
make install


# 输出安装完成, 同时等待一次键盘输入, 防止容器退出

echo "ebpf install success"
echo "press any key to exit"
read -r