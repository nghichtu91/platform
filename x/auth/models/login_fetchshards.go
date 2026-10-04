package models

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/opentracing/opentracing-go"

	"github.com/nghichtu91/platform/share/x/common/shard"

	"github.com/nghichtu91/platform/share/planx/util"
	"github.com/nghichtu91/platform/share/planx/virtual_gid"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/nghichtu91/platform/share/planx/etcd"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	"github.com/nghichtu91/platform/share/x/auth/config"
	"github.com/nghichtu91/platform/share/x/common/consts"
)

type ShardInfo struct {
	Sid       uint
	Gid       uint
	ShowState string
	StartTime int64 // 对外开放时间
}

type ShardInfoForClient struct {
	Name              string `json:"name"`
	DisplayName       string `json:"dn"`
	ShowState         string `json:"ss"`
	state             string
	sid               uint
	MultiLang         string `json:"multi_lang"`
	Lang              string `json:"lang"`
	StartTime         int64  `json:"st"`
	Recommend         int    `json:"recmd"`
	RecommendLanguage string `json:"recmd_lang"`
	VirtualGID        int    `json:"v_gid"`     // 虚拟大区ID
	GroupId           int    `json:"group_id"`  // 分组ID
	NameExtra         string `json:"nameExtra"` // 名称额外信息，某些特定分组会需要
}

func (info ShardInfoForClient) GetState() string {
	return info.state
}

func (info ShardInfoForClient) GetSID() uint {
	return info.sid
}

func (info ShardInfoForClient) Clone() ShardInfoForClient {
	return ShardInfoForClient{
		Name:              info.Name,
		DisplayName:       info.DisplayName,
		ShowState:         info.ShowState,
		state:             info.state,
		sid:               info.sid,
		MultiLang:         info.MultiLang,
		Lang:              info.Lang,
		StartTime:         info.StartTime,
		Recommend:         info.Recommend,
		RecommendLanguage: info.RecommendLanguage,
		VirtualGID:        info.VirtualGID,
		GroupId:           info.GroupId,
		NameExtra:         info.NameExtra,
	}
}

type ShardHasRole struct {
	Shard         string `json:"shard"`
	RoleName      string `json:"name"`
	RoleLevel     string `json:"level"`
	MainHero      string `json:"hero"`
	HistoryShard  string `json:"history_shard"`
	MainlineLevel string `json:"mainline_level"`
	HeadIcon      string `json:"head_icon"`
	ShowState     string `json:"ss"`
	LastLoginTime string `json:"llt"`
}

var (
	gShardsInfoLock sync.RWMutex
	gShardIdInfoMap map[string]ShardInfo         // sn -> gid
	gSidGidMap      map[string]string            // gid:sid -> gid
	gShardsInfoMap  map[int][]ShardInfoForClient // sid -> client
	gShardsInfoQuit chan struct{}
	gShardsInfoWait util.WaitGroupWrapper
)

func init() {
	gShardsInfoQuit = make(chan struct{}, 1)
}

func InitShardInfo() error {
	gid := fmt.Sprintf("%d", config.Cfg.CommonCfg.Gid)
	rev, err := UpdateAllShardsByGids([]string{gid}...)
	if err != nil {
		return err
	}

	// 开始监听变化
	serverKey := fmt.Sprintf("%s/%s/shards",
		config.Cfg.CommonCfg.EtcdServer, gid)
	etcd.WatchWithRevNoPrevRetry(serverKey, rev, true, gShardsInfoQuit, &gShardsInfoWait, func(resp clientv3.WatchResponse) {
		defer tilogs.PanicCatcher("InitShardInfo watch panic")
		// TODO 改为shard增量更新
		_, err := UpdateAllShardsByGids([]string{gid}...)
		if err != nil {
			tilogs.L().Errorf("InitShardInfo watch UpdateAllShardsByGids err: %s", err.Error())
			return
		}
	})
	return nil
}

func StopShardInfo() {
	close(gShardsInfoQuit)
	gShardsInfoWait.Wait()
}

// FetchShardsInfo 通过game id获取当前所有分服的明细
func FetchShardsInfo(gid int) ([]ShardInfoForClient, error) {
	gShardsInfoLock.RLock()
	defer gShardsInfoLock.RUnlock()
	sinfo, sok := gShardsInfoMap[gid]
	if !sok {
		return []ShardInfoForClient{}, errors.New(fmt.Sprintf("NoInfo gid %d", gid))
	}
	return sinfo[:], nil
}

// GetShardID 通过shardname获取shard id(sid)
func GetShardID(name string) (sid, gid uint, startTime int64, showState string, err error) {
	gShardsInfoLock.RLock()
	info, ok := gShardIdInfoMap[name]
	gShardsInfoLock.RUnlock()

	if !ok {
		return 0, 0, 0, "", errors.New(fmt.Sprintf("[GetShardID] shard %s NoInfo", name))
	}
	return info.Sid, info.Gid, info.StartTime, info.ShowState, nil
}

func GetShardName(sid, gid uint) (shardName string, err error) {
	gs_str := gid_sid_str(gid, sid)
	gShardsInfoLock.RLock()
	sh, ok := gSidGidMap[gs_str]
	gShardsInfoLock.RUnlock()
	if !ok {
		return "", errors.New(fmt.Sprintf("[GetShardName] sid %d gid %d NoInfo", sid, gid))
	}
	return sh, nil
}

// GetShardShowState 通过服务器名获取状态
func GetShardShowState(shardName string) (ss string, err error) {
	gShardsInfoLock.RLock()
	defer gShardsInfoLock.RUnlock()

	s, ok := gShardIdInfoMap[shardName]
	if ok {
		return s.ShowState, nil
	}
	return "", errors.New(fmt.Sprintf("[GetShardShowState] shardName %s NoInfo", shardName))
}

func GetUserShardHasRole(ctx context.Context, uid string) ([]ShardHasRole, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "GetUserShardHasRole")
	defer span.Finish()

	res, err := db_interface.GetUserShardInfo(uid)
	if err != nil {
		return nil, err
	}

	sort.Slice(res, func(i, j int) bool {
		// 登录时间靠后的排前面
		// 虽然这里的时间戳是string，但位数相等
		// 所以直接比较也能确保结果是对的
		return res[i].LastLoginTime > res[j].LastLoginTime
	})

	tilogs.L().Debugf("[GetUserShardHasRole] get uid %s from db info: %+v", uid, res)
	shs := make([]ShardHasRole, 0, len(res))
	for _, r := range res {
		if r.RoleLevel != "" {
			gid, sid, err := parse_gid_sid_str(r.GidSid)
			if err != nil {
				tilogs.L().Warnf("[GetUserShardHasRole] parse_gid_sid_str err %s", err.Error())
				continue
			}
			shardName, err := GetShardName(sid, gid)
			if err != nil {
				tilogs.L().Warnf("[GetUserShardHasRole] GetShardName err %s", err.Error())
				continue
			}
			ss, _ := GetShardShowState(shardName)
			shs = append(shs, ShardHasRole{
				Shard:         shardName,
				ShowState:     ss,
				RoleName:      r.RoleName,
				RoleLevel:     r.RoleLevel,
				MainHero:      r.MainHero,
				HistoryShard:  r.HistoryShard,
				MainlineLevel: r.MainlineLevel,
				HeadIcon:      r.HeadIcon,
				LastLoginTime: r.LastLoginTime,
			})
		}
	}
	tilogs.L().Debugf("[GetUserShardHasRole] GetRoles %+v", shs)
	return shs, nil
}

func SetUserShardHasRole(gid, sid int, uid string, roleName, roleLevel, mainHero, lastLoginTime, mainlineLevel, headIcon, playerId string) error {
	return db_interface.SetUserShardInfo(uid, gid_sid_str(uint(gid), uint(sid)), roleName, roleLevel, mainHero, lastLoginTime, mainlineLevel, headIcon, playerId)
}

func GetUserHistoryShard(uid string) ([]string, error) {
	return db_interface.GetUserHistoryShard(uid)
}

func SetUserHistoryShard(uid, shard string) error {
	return db_interface.SetUserHistoryShard(uid, shard)
}

func GetUserLastShard(ctx context.Context, uid string) (string, error) {
	span, _ := opentracing.StartSpanFromContext(ctx, "GetUserLastShard")
	defer span.Finish()

	gidsid, err := db_interface.GetLastLoginShard(uid)
	if err != nil || gidsid == "" {
		return "", err
	}
	gid, sid, err := parse_gid_sid_str(gidsid)
	if err != nil {
		tilogs.L().Warnf("[GetUserLastShard] parse_gid_sid_str err %s", err.Error())
		return "", err
	}
	shardName, err := GetShardName(sid, gid)
	if err != nil {
		tilogs.L().Warnf("[GetUserLastShard] GetShardName err %s", err.Error())
		return "", err
	}
	return shardName, nil
}

func SetUserLastShard(uid string, gid, sid int) error {
	return db_interface.SetLastLoginShard(uid, gid_sid_str(uint(gid), uint(sid)))
}

func SetUserPayFeedBackIfNot(uid string) (error, bool) {
	return db_interface.SetPayFeedBackIfNot(uid)
}

func gid_sid_str(gid, sid uint) string {
	return fmt.Sprintf("%d:%d", gid, sid)
}

func parse_gid_sid_str(gid_sid_str string) (gid, sid uint, err error) {
	ss := strings.Split(gid_sid_str, ":")
	if len(ss) < 2 {
		return 0, 0, errors.New(fmt.Sprintf("[parse_gid_sid_str] gid_sid err: %s", gid_sid_str))
	}
	g, err := strconv.Atoi(ss[0])
	if err != nil {
		return 0, 0, errors.New(fmt.Sprintf("[parse_gid_sid_str] err: %v", err))
	}
	s, err := strconv.Atoi(ss[1])
	if err != nil {
		return 0, 0, errors.New(fmt.Sprintf("[parse_gid_sid_str] err: %v", err))
	}
	return uint(g), uint(s), nil
}

func getShardsFromEtcdByGids(gids []string) ([]consts.ShardInfo, int64, error) {
	re := make([]consts.ShardInfo, 0, 512)
	var minRev int64 = math.MaxInt64 // 用于获取最小的rev
	for _, gid := range gids {
		intGid, err := strconv.Atoi(gid)
		if err != nil {
			return nil, 0, err
		}

		shardAllInfo, rev := shard.LoadShardInfoWithRev(config.Cfg.CommonCfg.EtcdRoot, config.Cfg.CommonCfg.EtcdServer, gid)
		if rev < minRev {
			minRev = rev
		}
		for key, sidInfo := range shardAllInfo {
			sid := parseShardStr(key)
			if sidInfo.SName == "" {
				continue
			}

			// 在服务器曾经成功启动过并操作过上线后，此服务器就可以在服务器列表中出现
			// etcd.StateOnline 这个检查由外面做，若是超级账号都能看到
			// || state != etcd.StateOnline

			shard := consts.ShardInfo{
				Gid:               uint(intGid),
				Sid:               sid,
				Ip:                sidInfo.Ip,
				SName:             sidInfo.SName,
				DName:             sidInfo.DisplayName,
				State:             sidInfo.State,
				ShowState:         fmt.Sprint(sidInfo.ShowState),
				StartTime:         sidInfo.PublicServerTime,
				Recommend:         sidInfo.RecommendValue,
				RecommendLanguage: sidInfo.RecommendLanguage,
				GroupId:           sidInfo.GroupId,
				NameExtra:         sidInfo.NameExtra,
			}
			if _shard_valid(shard) {
				re = append(re, shard)
			}
		}
	}
	return re, minRev, nil
}

func parseShardStr(path string) uint {
	s, err := strconv.Atoi(path)
	if err == nil {
		return uint(s)
	}
	return 0
}

func _shard_valid(s consts.ShardInfo) bool {
	_, err := strconv.Atoi(s.ShowState)
	if err != nil {
		_ = consts.ShowStateCount // 没配showstate的话，就给个不存在的值
	}

	return true
}

func UpdateAllShardsByGids(gids ...string) (int64, error) {
	shardsClientMap := make(map[int][]ShardInfoForClient, 16)
	shardsInfoMap := make(map[string]ShardInfo, 16)
	sidGidMap := make(map[string]string, 16)

	shards, rev, err := getShards(gids)
	if err != nil {
		return rev, err
	}
	tilogs.L().Debugf("getAllShards gids=%v, shards=%v", gids, shards)

	for _, shard := range shards {
		info, ok := shardsClientMap[int(shard.Gid)]
		if !ok {
			info = make([]ShardInfoForClient, 0, 32)
			shardsClientMap[int(shard.Gid)] = info
		}
		info = shardsClientMap[int(shard.Gid)]
		info = append(info, ShardInfoForClient{
			Name:              shard.SName,
			DisplayName:       shard.DName,
			ShowState:         shard.ShowState,
			state:             shard.State,
			sid:               shard.Sid,
			Lang:              shard.Language,
			MultiLang:         shard.MultiLang,
			StartTime:         shard.StartTime,
			Recommend:         shard.Recommend,
			RecommendLanguage: shard.RecommendLanguage,
			VirtualGID:        virtual_gid.GetVirtualGroupID(int(shard.Sid)),
			GroupId:           int(shard.GroupId),
			NameExtra:         shard.NameExtra,
		})
		shardsClientMap[int(shard.Gid)] = info

		shardsInfoMap[shard.SName] = ShardInfo{
			Sid:       shard.Sid,
			Gid:       shard.Gid,
			ShowState: shard.ShowState,
			StartTime: shard.StartTime,
		}
		sidGidMap[gid_sid_str(shard.Gid, shard.Sid)] = shard.SName
	}

	updateShardInfo(shardsClientMap, shardsInfoMap, sidGidMap)
	tilogs.L().Debugf("new shards info = %v, \n shardId = %v, \n nameInfo = %v", shardsInfoMap, sidGidMap, shardsClientMap)
	return rev, nil
}

func updateShardInfo(_gid_shards map[int][]ShardInfoForClient, _shardId_Info map[string]ShardInfo,
	_gid_sid_shardId map[string]string) {
	gShardsInfoLock.Lock()
	defer gShardsInfoLock.Unlock()

	if gShardIdInfoMap == nil {
		gShardIdInfoMap = _shardId_Info
	} else {
		for key, value := range _shardId_Info {
			gShardIdInfoMap[key] = value
		}
	}

	if gSidGidMap == nil {
		gSidGidMap = _gid_sid_shardId
	} else {
		for key, value := range _gid_sid_shardId {
			gSidGidMap[key] = value
		}
	}

	if gShardsInfoMap == nil {
		gShardsInfoMap = _gid_shards
	} else {
		for key, value := range _gid_shards {
			gShardsInfoMap[key] = value
		}
	}
	tilogs.L().Infof("update shard info gShardsInfoMap=%v gSidGidMap=%v gShardsInfoMap=%v", gShardsInfoMap, gSidGidMap, gShardsInfoMap)
}

func getShards(gids []string) ([]consts.ShardInfo, int64, error) {
	shards, rev, err := getShardsFromEtcdByGids(gids)
	if err != nil {
		return nil, rev, err
	}
	sort.Slice(shards, func(i, j int) bool {
		return shards[i].Sid < shards[j].Sid
	})
	return shards, rev, nil
}
