## 实现目的
目前极无双的网关是和Gamex在一个服务里的，缺点就是网络服务和游戏服务耦合在一起
因此将网络服务单独拆成网关，游戏服务是Gamex，优点就是玩家只连接网关，下层架构对玩家隔离，就有了进一步升级框架的可能

## 代码目录
vcs.taiyouxi.net/platform/x/gate

## 启动方式
目前此修改兼容已有的auth和gamex连接的模式不变，为了不影响极无双线上
新模式启动方式：
* auth的app.toml里gate_alone=true；启动依旧是 auth allinone
* gate启动方式：gate gate  (-c config.toml)
* gamex启动方式：gamex game (-c config.toml)

ps：启动顺序无要求

注意：

本地调试的时候gate的conf/gate.toml里的publicip需要改成本地ip

或者用 gate gate -c zgate.toml, 用另一个名字的git不跟踪的toml文件

## 修改内容
主要思路很简单，就是将原来网关和gamex之间chan连接改为用grpc连接
1. auth对gate，服务自动发现
2. gate和gamex之间，互相服务自动发现
3. gamex服务自动发现auth
4. auth和gamex之间已有的内网通信，改为了端口自动发现，涉及到了一系列配置和周边服务改动
5. 之前gate对应一个gamex，gate只有一个loginInfo用来保存从auth来的loginToken;
现在gate对应多个gamex，所以把loginInfo改为一个shard对应一个；
loginToken改为由之前的loginToken+sid拼接组成，方便gate知道找那个shard的loginInfo
6. 之前auth和其他服务之间的rpc调用，分配为：
  * gate->auth：notifylogin，notifylogout，（gateregister废弃了）
  * gamex->auth：notifyuserinfo
  * auth->gate：RegisterLoginToken，KickItOffline
7. gate和gamex之间，会建立10个grpc连接，玩家登陆的时候，会选择一个最小的ccu的连接建立stream；每个玩家就是一个stream
8. auth去掉了gateregister
  * 之前gamex用gateregister通知auth，自己的内网和外网地址
  * 现在是服务发现gate，并在进入游戏时检查下gamex状态

ps: 其中3、4，在老版本中也进行了修改

## 服务发现
* auth
  * auth启动后，自动发现一个可用的端口，并将内网地址和端口、以及url关键字注册到etcd(action：set no ttl)；没有做关服删除注册的内容（如果auth关闭了，通知不到就通知不到吧）
  * auth启动后开启etcd watch，实时监听gate注册目录的变化
    * 外网地址和内外地址（action：create, expire, delete）
    * 上线标志online（action：set）
    * ccu（action：set, expire, delete）
* gate
  * gate启动后，自动发现一个可用的端口，作为auth通知他authtoken的内网rpc用
  * gate启动后开启etcd watch，实时监视auth注册目录的变化（action：set）
  * gate定时注册ccu，内外和外网地址到etcd；关服时删除
    * 外网地址和内外地址（action：create, set(ttl), expire, delete）
    * ccu（action：set(ttl), expire, delete）
  * gate启动后开启etcd watch，实时监听gamex注册目录的变化
    * 内网地址（action：set, delete）
* gamex
  * gamex启动后，自动发现一个可用的端口，作为gate用grpc连接用（gamex作为server端）
  * gamex注册etcd
    * 内网地址，关服删除（action：set, delete）；没有做set的ttl
    如果gamex非正常关服，gate和gamex之间的链接也会断开，在线玩家会断开; 新登录的会报100错误
    * 已有模式，内网ip还是定时注册到etcd（action：set(ttl)）
  * gamex启动后开启etcd watch，实时监视auth注册目录的变化（action：set）
 

## metric
  * gamex
    gamex.[ip].ccu，这个是目前要关心的ccu了，名字和以前不一样了，需要东平那修改
  * gate
    gate.[ip].ccu

## 测试覆盖
账号互踢
增加和减少gate
增加gamex
gm工具，封禁相关
gift_sender，祈愿

## conf(toml)改动项
auth加gate_alone, internalip
gamex加internalip，sync_ip_tick

## gm工具工作
gate需要做单独的管理页，设置gate的上线标志（在etcd中）；并查看gate的各种状态
gamex的shard管理页，外网ip列可以去掉了

## 几个潜规则
  * 服务器发现在启动的时候，基本都是先拿指定目录下所有值再进行监听，若这两步中间有什么变化，可能会漏掉；没想到太好的办法，先用gm工具人工检查保证吧
  * 由于gate会保证一个acid同时只能登录进来一个，所以在gamex上没考虑同个acid同时活跃的保护

## 不够完善的点
  * 目前gamex已有的etcd注册的键，不好动了，还是应该把有ttl和没有的分开，auth也好监听，目前登录是每个请求去查一次，ccu大了还是有危险


## 和客户端对接
有踢人的IDS需要客户端识别一下
  * IDS_LOGIN_TIPS_KICKNOTIFY
  * IDS_ERROR_NETWORK_90000