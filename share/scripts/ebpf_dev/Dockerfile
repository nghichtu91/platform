# 内核源码的底板镜像
FROM docker/for-desktop-kernel:${this_is_error_you_must_change_it} AS kernel

# 运行时镜像
FROM centos:centos7.9.2009

WORKDIR /
COPY --from=kernel /kernel-dev.tar /
RUN tar xf kernel-dev.tar && rm kernel-dev.tar

# 复制部分文件到容器中
COPY hello_world.py cmake.patch ebpf-enable.sh build.sh /root/

WORKDIR /root
CMD mount -t debugfs debugfs /sys/kernel/debug && /bin/bash