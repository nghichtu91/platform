package merge_db

import (
	"errors"
	"strconv"

	"github.com/nghichtu91/platform/share/x/common/msg/merger/pb"

	"github.com/gomodule/redigo/redis"

	"github.com/nghichtu91/platform/share/planx/metrics"
	"github.com/nghichtu91/platform/share/planx/redispool"
)

var (
	ErrCBRepliesLenNotMatch = errors.New("replies length from command buffer is not match") // redis返回数据长度和预期不一致
)

// ICMDLoader 用于redis command buffer的接口
// db代码生成工具里已经实现了下列两个接口
// 可以使用redis command buffer一次读取多个db，提升性能
type ICMDLoader interface {
	// SendLoadCMD
	// 将读取db的命令写入cb中
	SendLoadCMD(conn redis.Conn) error

	// UnmarshalFromReply
	// 将执行完毕后的cb结果读取回内存中
	UnmarshalFromReply(reply interface{}) error
}

type IProtoDB interface{}

// MergeSet 一组用于合服的信息
// 包括db以及对应的redisPool
type MergeSet struct {
	// Pool DB使用的redis链接
	// 单独放出来是为了合并相同db的读取，用cb批量实现
	Pool redispool.IPool

	// DB
	// 需要进行合并或者新生成的表
	DB IMergeDB

	// RenamePools
	// 用于处理改名请求的pool
	// 譬如检查重名时，需要在Names表内获取acid，然后拉取BaseInfo表进行改名操作
	// 被合的服不需要这个值
	RenamePools []redispool.IPool

	// RollbackKeys 回滚数据的Redis Keys
	// 进行合服时，如果需要直接修改当前数据，必须对旧数据保存以便回滚
	// 回滚数据类型为HashMap，hash key为acid
	RollbackKeys []string

	// RenameDB
	// 使用时需要转回protopb.Rename
	// 记录名字修改并在后续gamex启动后统一推送其他模块处理
	// 被合的服不需要这个值
	RenameDB ICMDLoader

	// 合服清理条件
	*CleanOption

	// MergeReq 合并计划
	MergeReq *pb.StartMergeReq

	NamePrefix DuplicatePrefix
}

// IMergeDB 合服数据库合并接口
// 只有toml定义中，need_merge=true的模块会实现SaveAfterMerge
type IMergeDB interface {
	ICMDLoader

	// SaveAfterMerge
	// 合服逻辑处理完成后，将新数据写回
	SaveAfterMerge() error
}

// BatchLoad 批量读取数据
func BatchLoad(pool redispool.IPool, loaders []ICMDLoader, metricsPrefix string) error {
	conn := pool.GetDBConn()
	defer conn.Close()

	for _, m := range loaders {
		if err := m.SendLoadCMD(conn.Conn); err != nil {
			return err
		}
	}

	replies, err := redis.Values(conn.DoCmdBuffer(metricsPrefix, false))
	if err != nil {
		return err
	}
	if len(replies) != len(loaders) {
		return ErrCBRepliesLenNotMatch
	}

	for i := range replies {
		if replies[i] == nil {
			continue
		}
		if err := loaders[i].UnmarshalFromReply(replies[i]); err != nil {
			return err
		}
	}

	return nil
}

// MergeOpFunc 合并数据库操作
type MergeOpFunc func(tar *MergeSet, src []IMergeDB) error

// MergeDB 合服操作
// 从merges读取旧数据
// 执行mergeOP，即合服时需要做的额外操作，譬如合并数据库，删除特定值等
// mergeOP中，tar是合服后的DB，src是被合的所有DB
// 如果newMerge.DB不为nil，需要保存新生成的数据
func MergeDB(mergeOP MergeOpFunc, newMerge *MergeSet, merges ...*MergeSet) error {
	dbs := make([]IMergeDB, 0, len(merges))
	batches := make(map[redispool.IPool][]ICMDLoader)

	// 读取各自的数据
	for _, m := range merges {
		batches[m.Pool] = append(batches[m.Pool], m.DB)
		dbs = append(dbs, m.DB)
	}

	for pool, mdb := range batches {
		if err := BatchLoad(pool, mdb, metrics.GetDBStatPrefix("gmserver", "LoadBeforeMerge", "DoCmdBuffer")); err != nil {
			return err
		}
	}

	// 执行合服逻辑
	if err := mergeOP(newMerge, dbs); err != nil {
		return err
	}

	if newMerge.DB != nil {
		return newMerge.DB.SaveAfterMerge()
	}

	return nil
}

// CleanOption 合服清除条件
type CleanOption struct {
	// 是否开启合服清理账号
	Enable bool

	// RechargeAmount 充值额度，大于此金额的账号不会被删除
	RechargeAmount uint64

	// Level 等级，大于此数字的账号不删除
	Level uint32

	// LastLoginTS 最后一次登录时间戳，大于此数字的账号不删除
	LastLoginTS int64
}

type DuplicatePrefix struct {
	PlayerPrefix string
	GuildPrefix  string
}

// GetPlayerName 获取玩家名字
func (dp *DuplicatePrefix) GetPlayerName(playerID int64) string {
	return dp.PlayerPrefix + strconv.Itoa(int(playerID))
}

// GetGuildName 获取公会名字
func (dp *DuplicatePrefix) GetGuildName(guildID int64) string {
	return dp.GuildPrefix + strconv.Itoa(int(guildID))
}
