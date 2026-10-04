package user_info

import (
	"sync"

	"github.com/nghichtu91/platform/share/planx/redispool"
)

var (
	// 玩家信息
	user UserInfo
	// 可以把连接信息也写在这
	connInfo ConnectInfo
	// 启动多个连接时的数据
	multiInfo MultiInfo
	// 大区id
	gid int
	// 聊天服db
	chatDb     redispool.IPool
	chatDbLock sync.RWMutex
	// 机器人的信息
	robots    map[int]*Robot
	robotLock sync.RWMutex
)

func init() {
	robots = make(map[int]*Robot, 64)
}

type UserInfo struct {
	// 对应玩家的唯一id
	acid  string
	token string
}

type ConnectInfo struct {
	addr string
}

type MultiInfo struct {
	startId int
	count   int
}

type Robot struct {
	UserInfo
}

func InitUser(acid string) {
	user.acid = acid
}

func InitConnect(addr string) {
	connInfo.addr = addr
}

func GetUserAcid() string {
	return user.acid
}

func GetCometAddr() string {
	return connInfo.addr
}

func GetUserToken() string {
	return user.token
}

func SetUserToken(token string) {
	user.token = token
}

func GetGid() int {
	return gid
}

func SetGid(id int) {
	gid = id
}

func SetMultiStartId(startId int) {
	multiInfo.startId = startId
}

func GetMultiStartId() int {
	return multiInfo.startId
}

func SetMultiCount(count int) {
	multiInfo.count = count
}

func GetMultiCount() int {
	return multiInfo.count
}

func IsMultiple() bool {
	return multiInfo.count > 1
}

func SetChatDb(db redispool.IPool) {
	chatDb = db
}

func GetChatDb() redispool.IPool {
	return chatDb
}

func AddRobot(acid, token string, id int) {
	robotLock.Lock()
	robots[id] = &Robot{
		UserInfo{acid, token},
	}
	robotLock.Unlock()
}

func GetRobot(id int) *Robot {
	robotLock.RLock()
	r, _ := robots[id]
	robotLock.RUnlock()
	return r
}

func (r *Robot) GetUserId() string {
	return r.acid
}
func (r *Robot) GetToken() string {
	return r.token
}
