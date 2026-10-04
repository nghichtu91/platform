package dao

import (
	"errors"
	"strings"

	"github.com/nghichtu91/platform/share/x/chat/logicx/logic/silence_sys"

	"github.com/nghichtu91/platform/share/planx/metrics"

	"github.com/golang/protobuf/proto"
	"github.com/gomodule/redigo/redis"
	chat_common "github.com/nghichtu91/platform/share/planx/servers/chat"
	log "github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/chat/api/const_value"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
	c "github.com/nghichtu91/platform/share/x/chat/logicx/config"
)

const ActivityChannel = "ActivityChannel"

// GetPlayerToken 获取登录token
func (dao *Dao) GetPlayerToken(userId string) (string, error) {
	_db := dao.redis.Get()
	defer _db.Close()

	key := chat_common.GetTokenKey(userId)
	data, err := redis.String(_db.Do(metrics.GetDBStatPrefix("chat", "GetPlayerToken", "GET"), "GET", key))
	if err != nil {
		log.L().Infof("get player token failed, userId:%s, error:%v", userId, err)
		return "", err
	}

	return data, nil
}

// SetPlayerMapping 登录成功后设置user与comet+sessionId的映射
func (dao *Dao) SetPlayerMapping(userId, data string) error {
	_db := dao.redis.Get()
	defer _db.Close()

	key := chat_common.GetMappingKey(userId)
	_, err := redis.String(_db.Do(metrics.GetDBStatPrefix("chat", "SetPlayerMapping", "SET"), "SET", key, data))
	if err != nil {
		return err
	}
	return nil
}

// DelPlayerMapping 删除映射关系,私聊就不可用了,因为找不到对应的comet去发
func (dao *Dao) DelPlayerMapping(userId, sessionId string) error {
	_db := dao.redis.Get()
	defer _db.Close()

	// lua脚本,先查后刪保证原子
	luaScript := redis.NewScript(1, `
		local key, session = KEYS[1],ARGV[1]
		local data = redis.call('get',key)
		if data ~= false then
			local index, _ = string.find(data, ",")
			local dbResult = string.sub(data, index+1, string.len(data))
			if dbResult == session then
        		redis.call('del',key)
    		end
		end
	`)
	luaScript.Load(_db.Conn)
	_, err := luaScript.Do(_db.Conn, chat_common.GetMappingKey(userId), sessionId)
	return err
}

// GetPlayerMapping 获得映射关系
func (dao *Dao) GetPlayerMapping(userId string) (string, error) {
	_db := dao.redis.Get()
	defer _db.Close()

	key := chat_common.GetMappingKey(userId)
	data, err := redis.String(_db.Do(metrics.GetDBStatPrefix("chat", "GetPlayerMapping", "GET"), "GET", key))
	if err != nil {
		log.L().Infof("get player mapping failed, userId:%s, error:%v", userId, err)
		return "", err
	}

	slices := strings.Split(data, ",")
	if len(slices) != 2 {
		log.L().Warnf("get player mapping failed, split data failed, userId:%s, data:%s", userId, data)
		return "", errors.New(pb.ErrorCode_RedisTargetDataError.String())
	}

	return data, nil
}

// SavePersonalMsg 保存私聊的消息
func (dao *Dao) SavePersonalMsg(from, target string, data *pb.ChatMsgData) (err error) {
	var (
		curMsgCount int
		trimRet     string
		msg         []byte
	)
	_db := dao.redis.Get()
	defer _db.Close()

	msg, err = proto.Marshal(data)
	if err != nil {
		log.L().Errorf("save personal msg marshal failed, from:%s, target:%s, data:%v", from, target, data)
		return
	}

	key := chat_common.GetPersonalKey(target)
	curMsgCount, err = redis.Int(_db.Do(metrics.GetDBStatPrefix("chat", "SavePersonalMsg", "LPUSH"), "LPUSH", key, msg))
	if err != nil {
		log.L().Errorf("save personal msg redis failed, from:%s, target:%s, error:%v", from, target, err)
		return
	}

	log.L().Debugf("save personal msg success, current count:%d", curMsgCount)
	if curMsgCount > c.Cfg.Personal.MaxPersonSaveCount {
		trimRet, err = redis.String(_db.Do(metrics.GetDBStatPrefix("chat", "TrimPersonalMsg", "LTRIM"),
			"LTRIM", key, 0, c.Cfg.Personal.MaxPersonSaveCount))
		if err != nil {
			log.L().Errorf("save personal msg ltrim redis failed, from:%s, target:%s, count:%d, error:%v", from, target, curMsgCount, err)
			return
		}
		log.L().Debugf("trim personal msg success, result:%s", trimRet)
	}

	_, err = redis.Int(_db.Do(metrics.GetDBStatPrefix("chat", "ExpirePlayerMapping", "EXPIRE"),
		"EXPIRE", key, c.Cfg.Personal.MaxPersonKeySaveTime))
	if err != nil {
		log.L().Errorf("save personal msg expire failed, from:%s, target:%s, error:%v", from, target, err)
		return err
	}
	return nil
}

// SaveRoomMsg room的配置可能会更多样化一些 每种类型的room配置不同 所以单独处理
func (dao *Dao) SaveRoomMsg(roomId string, data *pb.ChatMsgData) (err error) {
	var (
		curMsgCount int
		trimRet     string
		msg         []byte
	)
	_db := dao.redis.Get()
	defer _db.Close()

	roomType := const_value.GetRoomType(roomId)
	roomSaveCount := c.GetRoomSaveHistoryMsgCount(roomType)
	if roomSaveCount <= 0 {
		// 如果聊天消息不需要存储的话 直接返回
		return
	}

	msg, err = proto.Marshal(data)
	if err != nil {
		log.L().Errorf("save room msg marshal failed, room:%s, data:%v", roomId, data)
		return
	}

	key := chat_common.GetRoomSaveKey(roomId)
	curMsgCount, err = redis.Int(_db.Do(metrics.GetDBStatPrefix("chat", "SaveRoomMsg", "LPUSH"), "LPUSH", key, msg))
	if err != nil {
		log.L().Errorf("save room msg redis failed, room:%s, err:%v", roomId, err)
		return
	}

	log.L().Debugf("save room msg success, key:%s current count:%d, limit count: %d", key, curMsgCount, roomSaveCount)
	if curMsgCount > roomSaveCount {
		trimRet, err = redis.String(_db.Do(metrics.GetDBStatPrefix("chat", "TrimRoomMsg", "LTRIM"),
			"LTRIM", key, 0, roomSaveCount))
		if err != nil {
			log.L().Errorf("save room msg ltrim redis failed, room:%s, count:%d err:%v", roomId, curMsgCount, err)
			return
		}
		log.L().Debugf("trim room msg success, result:%s", trimRet)
	}

	if strings.Contains(key, ActivityChannel) {
		_, err = redis.Int(_db.Do(metrics.GetDBStatPrefix("chat", "ExpireRoomKey", "EXPIRE"),
			"EXPIRE", key, c.GetActivityRoomStoreMsgTime(roomType)))
		if err != nil {
			log.L().Errorf("save room msg expire redis failed, room:%s, err:%v", roomId, err)
			return err
		}
	} else {
		_, err = redis.Int(_db.Do(metrics.GetDBStatPrefix("chat", "ExpireRoomKey", "EXPIRE"),
			"EXPIRE", key, c.GetRoomStoreMsgTime(roomType)))
		if err != nil {
			log.L().Errorf("save room msg expire redis failed, room:%s, err:%v", roomId, err)
			return err
		}
	}

	return nil
}

// FetchPersonMsg 获取私聊消息
func (dao *Dao) FetchPersonMsg(target string, begin, end uint32) (his []*pb.ChatMsgData, err error) {
	var key string
	_db := dao.redis.Get()
	defer _db.Close()

	key = chat_common.GetPersonalKey(target)
	msgBytesList, err := redis.ByteSlices(_db.Do(metrics.GetDBStatPrefix("chat", "FetchMsg", "LRANGE"), "LRANGE", key, begin, end))
	if err != nil {
		log.L().Errorf("fetch personal history error. target:%s, begin:%d, end:%d err=%v", target, begin, end, err)
		return nil, err
	}

	if msgBytesList == nil || len(msgBytesList) == 0 {
		log.L().Debugf("fetch personal history empty. target:%s, begin:%d, end:%d err=%v", target, begin, end, err)
		return nil, nil
	}

	filterHis := make(map[string]string, len(msgBytesList))
	hisTmp := make([]*pb.ChatMsgData, 0, len(msgBytesList))
	for _, msgBytes := range msgBytesList {
		msg := &pb.ChatMsgData{}
		err = proto.Unmarshal(msgBytes, msg)
		if err != nil {
			log.L().Errorf("unmarshal offline msg error. target:%s, err=%v", target, err)
			return nil, err
		}
		filterHis[msg.GetUuid()] = msg.GetFromId()
		hisTmp = append(hisTmp, msg)
	}

	filterHis = silence_sys.GetModule().GetPlayerSilenceSys(filterHis)

	for _, hisT := range hisTmp {
		if _, ok := filterHis[hisT.GetUuid()]; ok {
			his = append(his, hisT)
		}
	}

	_, err = redis.Int(_db.Do(metrics.GetDBStatPrefix("chat", "ExpirePlayerMapping", "EXPIRE"),
		"EXPIRE", key, 0))
	if err != nil {
		log.L().Debugf("fetch personal history expire faild. target:%s, begin:%d, end:%d err=%v", target, begin, end, err)
		return nil, err
	}
	log.L().Debugf("fetch msg success, fetch count:%d", len(his))
	return his, nil
}

// FetchRoomMsg 获取房间消息
func (dao *Dao) FetchRoomMsg(target string, begin, end uint32) (his []*pb.ChatMsgData, err error) {
	var key string
	_db := dao.redis.Get()
	defer _db.Close()
	//his = make([]*pb.ChatMsgData, 0)

	key = chat_common.GetRoomSaveKey(target)
	msgBytesList, err := redis.ByteSlices(_db.Do(metrics.GetDBStatPrefix("chat", "FetchMsg", "LRANGE"), "LRANGE", key, begin, end))
	if err != nil {
		log.L().Errorf("fetch room history error. key:%s, err=%v", key, err)
		return nil, err
	}

	if msgBytesList == nil || len(msgBytesList) == 0 {
		log.L().Debugf("history empty. key:%s, err=%v", key, err)
		return nil, nil
	}

	filterHis := make(map[string]string, len(msgBytesList))
	hisTmp := make([]*pb.ChatMsgData, 0, len(msgBytesList))
	for _, msgBytes := range msgBytesList {
		msg := &pb.ChatMsgData{}
		err = proto.Unmarshal(msgBytes, msg)
		if err != nil {
			log.L().Errorf("unmarshal offline msg error. key:%s, err=%v", key, err)
			return nil, err
		}

		filterHis[msg.GetUuid()] = msg.GetFromId()
		hisTmp = append(hisTmp, msg)
	}

	filterHis = silence_sys.GetModule().GetPlayerSilenceSys(filterHis)

	for _, hisT := range hisTmp {
		if _, ok := filterHis[hisT.GetUuid()]; ok {
			his = append(his, hisT)
		}
	}

	// todo roomId未确认
	if strings.Contains(key, ActivityChannel) {
		_, err = redis.Int(_db.Do(metrics.GetDBStatPrefix("chat", "ExpireRoomKey", "EXPIRE"),
			"EXPIRE", key, c.GetActivityRoomStoreMsgTime(target)))
		if err != nil {
			log.L().Errorf("expire room key error. key:%s, err=%v", key, err)
			return nil, err
		}
	} else {
		_, err = redis.Int(_db.Do(metrics.GetDBStatPrefix("chat", "ExpireRoomKey", "EXPIRE"),
			"EXPIRE", key, c.GetRoomStoreMsgTime(target)))
		if err != nil {
			log.L().Errorf("expire room key error. key:%s, err=%v", key, err)
			return nil, err
		}
	}

	log.L().Debugf("fetch room msg success, fetch count:%d", len(his))
	return his, nil
}

/*
	设置禁言标记 禁言只有在平台设置的时候才保存
	只有在上线的时候才会读一次禁言标记
*/
func (dao *Dao) SetPlayerForbidden(userId, reason string, endTime int64) (err error) {
	var buf []byte
	_db := dao.redis.Get()
	defer _db.Close()

	key := chat_common.GetForbiddenKey(userId)
	data := &pb.SaveForbiddenInfo{
		Reason:  &reason,
		EndTime: &endTime,
	}

	if buf, err = proto.Marshal(data); err != nil {
		log.L().Errorf("marshal forbidden msg error. key:%s, err=%v", key, err)
		return
	}

	_, err = redis.String(_db.Do(metrics.GetDBStatPrefix("chat", "SetPlayerForbidden", "SET"), "SET", key, buf))
	if err != nil {
		log.L().Errorf("save forbidden msg error. key:%s, err=%v", key, err)
		return
	}

	log.L().Debugf("SetPlayerForbidden successful, key:%s, reason:%s, endTime:%d", key, reason, endTime)
	return nil
}

// DelPlayerForbidden 删除映射关系 私聊就不可用了 因为找不到对应的comet去发
func (dao *Dao) DelPlayerForbidden(userId string) error {
	_db := dao.redis.Get()
	defer _db.Close()

	key := chat_common.GetForbiddenKey(userId)
	_, err := redis.Int64(_db.Do(metrics.GetDBStatPrefix("chat", "DelPlayerForbidden", "DEL"), "DEL", key))
	if err != nil {
		log.L().Errorf("del forbidden msg error. key:%s, err=%v", key, err)
		return err
	}

	log.L().Debugf("DelPlayerForbidden successful, key:%s", key)
	return nil
}

// GetPlayerForbidden 获得禁言信息
func (dao *Dao) GetPlayerForbidden(userId string) (string, int64, error) {
	_db := dao.redis.Get()
	defer _db.Close()

	key := chat_common.GetForbiddenKey(userId)
	dbData, err := redis.Bytes(_db.Do(metrics.GetDBStatPrefix("chat", "GetPlayerForbidden", "GET"), "GET", key))
	if err != nil {
		return "", 0, err
	}

	pbData := &pb.SaveForbiddenInfo{}
	err = proto.Unmarshal(dbData, pbData)
	if err != nil {
		log.L().Errorf("get forbidden unmarshal msg error. key:%s, err=%v", key, err)
		return "", 0, err
	}
	return pbData.GetReason(), pbData.GetEndTime(), nil
}

func (dao *Dao) RoomDestroy(roomIds []string) (err error) {
	var key string
	_db := dao.redis.Get()
	defer _db.Close()

	for _, roomId := range roomIds {
		key = chat_common.GetRoomSaveKey(roomId)
		_, err := redis.Int64(_db.Do(metrics.GetDBStatPrefix("chat", "DelPlayerMapping", "DEL"), "DEL", key))
		if err != nil {
			log.L().Errorf("delete room cache error. key:%s, err=%v", key, err)
			return err
		}

		log.L().Debugf("delete room and history msg. key:%s, err=%v", key, err)
	}

	return nil
}
