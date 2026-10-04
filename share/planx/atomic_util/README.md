# atomic_util
需要跨服务进行原子性操作的组件

其中保证原子性的部分，是对应数据库保证的

目前可选的持久化数据库：
* redis
* mongo
* mysql
* ~~etcd~~