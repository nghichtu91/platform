# 根据输入选择
# build 构建镜像
# remove 删除镜像
# run 首次启动容器
# install 安装bcc开发环境
# start 启动容器
# delete 删除容器
# stop 停止容器
# exec 进入容器

function usage() {
    echo "Usage: $0 [build|remove|run|start|stop|delete|exec]"
    echo "build 构建镜像"
    echo "remove 删除镜像"
    echo "run 首次启动容器, 同时会尝试安装bcc开发环境(等价于自动install)"
    echo "install 安装bcc开发环境. 你要非得安装多次也行, 我也拦不住你"
    echo "start 启动容器"
    echo "delete 删除容器"
    echo "stop 停止容器"
    echo "exec 进入容器"
    echo ""
    echo "注意:"
    echo " run和start的区别是run是首次启动容器, start是启动已经停止的容器"
    
    exit 1
}

# 获取输入并判断是否合法
if [ $# -ne 1 ]; then
    usage
fi


IMAGE_NAME="ebpf-centos7"
CONTAINER_NAME="ebpf_container_centos7"


# docker 自带的输出就够了, 这里的检查有点多余了
# 镜像相关的操作得需要拦截一下, 避免误操作

# check_image. 参数可以输入1/0. 0表示存在就退出, 1表示不存在就退出
function check_image() {

  # 判断参数是否为0或者1
  if [ "$1" -ne 0 ] && [ "$1" -ne 1 ]; then
    echo "check_image函数的参数只能为0或者1"
    exit 1
  fi

  docker images | grep "$IMAGE_NAME" > /dev/null 2>&1
  if [ "$?" -eq "$1" ]; then
    echo "镜像${IMAGE_NAME}不满足检查${1}条件"
    exit 1
  fi
}

# check_container. 参数可以输入1/0. 0表示存在就退出, 1表示不存在就退出
function check_container() {
  return 0

  # 判断参数是否为0或者1
  if [ "$1" -ne 0 ] && [ "$1" -ne 1 ]; then
    echo "check_container函数的参数只能为0或者1"
    exit 1
  fi

  docker ps -a | grep "$CONTAINER_NAME" > /dev/null 2>&1
  if [ "$?" -eq "$1" ]; then
    echo "容器${CONTAINER_NAME}不满足检查${1}条件"
    exit 1
  fi
}

# check_container_state. 参数0表示容器运行, 1表示容器停止
function check_container_state() {
  return 0

  # 判断参数是否为0或者1
  if [ "$1" -ne 0 ] && [ "$1" -ne 1 ]; then
    echo "check_container_state函数的参数只能为0或者1"
    exit 1
  fi

  docker ps | grep "$CONTAINER_NAME" > /dev/null 2>&1
  if [ "$?" -eq "$1" ]; then
    echo "容器${CONTAINER_NAME}运行状态不满足检查${1}条件"
    exit 1
  fi
}

# 基于输入执行操作

if [ "$1" = "build" ]; then
    check_image 0 # 镜像存在就退出
    echo "build"
    docker build -t "$IMAGE_NAME" .
elif [ "$1" = "remove" ]; then
    check_image 1 # 镜像不存在就退出
    check_container 0 # 容器存在就退出

    echo "remove"
    docker rmi "$IMAGE_NAME"
elif [ "$1" = "run" ]; then
    check_image 1 # 镜像不存在就退出
    check_container 0 # 容器存在就退出

    echo "run"
    docker run --interactive \
          --tty \
          --detach \
          --name="$CONTAINER_NAME" \
          --privileged \
          -v /lib/modules:/lib/modules:ro \
          -v /etc/localtime:/etc/localtime:ro \
          --pid=host "$IMAGE_NAME" || exit 1
    check_container_state 1 # 容器已经停止就退出
    # 进入容器执行安装脚本
    docker exec -it "$CONTAINER_NAME" bash /root/build.sh
elif [ "$1" = "install" ]; then
    check_container 1 # 容器不存在就退出
    check_container_state 1 # 容器已经停止就退出

    echo "install"
    docker exec -it "$CONTAINER_NAME" bash /root/build.sh
elif [ "$1" = "start" ]; then
    check_container 1 # 容器不存在就退出
    check_container_state 0 # 容器正在运行就退出

    echo "start"
    docker start "$CONTAINER_NAME" || exit 1
    docker exec -it "$CONTAINER_NAME" /bin/bash
elif [ "$1" = "stop" ]; then
    check_container 1 # 容器不存在就退出
    check_container_state 1 # 容器已经停止就退出

    echo "stop"
    docker stop "$CONTAINER_NAME"
elif [ "$1" = "delete" ]; then
    check_container 1 # 容器不存在就退出
    check_container_state 0 # 容器正在运行就退出

    echo "delete"
    docker rm "$CONTAINER_NAME"
elif [ "$1" = "exec" ]; then
    check_container 1 # 容器不存在就退出
    check_container_state 1 # 容器已经停止就退出

    echo "exec"
    docker exec -it "$CONTAINER_NAME" /bin/bash
else
    usage
fi