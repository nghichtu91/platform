#JWS2-GM

   基于react，ant-design，dva，mysql, golang



##宗旨与目的

- 期望打造一套基于react，ant-design，dva于一体的、企业级后台管理系统

- 期望可以单纯由前端来解决用户权限，后端提供权限数据支持的一套完善的权限管理功能后台管理系统

- 期望可以在antd与dva的基础上，再次封装简单且可复用的基类组件，方便使用者简单接入，简单使用，简单拓展

##特性

- 基于react，ant-design，dva企业级后台管理系统最佳实践

- 基于Antd UI 设计语言，提供后台管理系统常见使用场景

- 浅度响应式设计

- webpack打包处理路由时，实现Javascript模块化按需动态dynamic加载



##目录结构


    ├── /dist/                 # 项目输出目录
    ├── /src/                  # 项目源码目录
    │ ├── /components/         # 项目组件
    │ │ ├── /common/           # 项目公共组件
    │ ├── /routes/             # 路由组件
    │ ├── /models/             # 数据模型
    │ ├── /services/           # 数据接口
    │ ├── /utils/              # 工具函数
    │ ├── router-dynamic.js    # 路由配置
    │ ├── index.js             # 入口文件
    │ └── index.html
    └── package.json           # 项目信息



##快速开始

#####进入目录安装依赖:

    npm install 或者 yarn 或者 yarn install



##开发：

    golang端:进入 /threekingdom/src/vcs.taiyouxi.net/platform/x/gm_tools 目录下

    首先go build 然后 gm_tools allinone

    web端:/threekingdom/src/vcs.taiyouxi.net/platform/x/gm_tools/webapp 目录下

    npm run watch




##构建：

    npm run build-dev        #本地环境发布
    npm run build-staging    #staging 环境发布
    npm run build-release    #release 环境发布
    build后的文件将会生成dist目录



##注意事项
	todo




##参考

    用户列表：https://github.com/dvajs/dva-example-user-dashboard
    zuiidea: https://github.com/zuiidea/antd-admin
    sorrycc: [https://github.com/dvajs/dva-example-user-dashboard]


