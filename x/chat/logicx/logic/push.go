package logic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nghichtu91/platform/share/planx/metrics"
	conf "github.com/nghichtu91/platform/share/x/chat/jobx/config"
	"github.com/nghichtu91/platform/share/x/chat/logicx/logic/translate"

	"github.com/opentracing/opentracing-go"

	"github.com/golang/protobuf/proto"
	uuid "github.com/satori/go.uuid"
	log "github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/tilogs/chatlog"
	"github.com/nghichtu91/platform/share/x/chat/api/const_value"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
	"github.com/nghichtu91/platform/share/x/chat/logicx/config"
	"github.com/nghichtu91/platform/share/x/chat/logicx/logiclog"
)

const PersonalHistoryPerTime = 5

// push to target
func (l *Logic) PushKey(c context.Context, fatherSpan opentracing.Span, msg *pb.GamexChatMsg, sendTime int64) (err error) {
	var (
		resp *pb.ClientPackage
	)
	log.L().Infof("logic PushKey fromId:%s targetId:%s, content:%s, bubble: %d", msg.GetFromId(), msg.GetTargetId(), msg.GetContent(), msg.GetBubble())

	// raw data
	rawdata := &pb.ChatMsgData{
		Uuid:         proto.String(uuid.NewV4().String()),
		FromId:       msg.FromId,
		TargetId:     msg.TargetId,
		Type:         pb.ChatType_Personal.Enum(),
		Content:      msg.Content,
		SendStamp:    &sendTime,
		SdkParam:     msg.SdkParam,
		TemplateInfo: msg.TemplateInfo,
		ExtraParam:   msg.ExtraParam,
		TrumtType:    msg.TrumtType,
		VoiceChatUrl: msg.VoiceChatUrl,
		Bubble:       msg.Bubble,
	}

	r := &pb.ChatMsgResp{
		Msg:  rawdata,
		Code: pb.ErrorCode_Success.Enum(),
	}
	resp, err = generateChatMsgResp(r)
	if err != nil {
		log.L().Warnf("logic pushkey, generateChatMsgResp failed, fromId:%s targetId:%s, conetent:%s", msg.GetFromId(), msg.GetTargetId(), msg.GetContent())
		return
	}

	// platform log
	chatlog.LogChatPlatform(
		logiclog.CommonInfo(
			msg.GetFromId(), // operator
			chatlog.PersonalChat,
			false,           // 房间消息
			false,           // 是否是喇叭消息
			msg.GetFromId(), // sender
			msg.GetExtraParam().GetPlayerShowInfo().GetName(),
			msg.GetTargetId(),
			"",               // device
			msg.GetContent(), // 文本内容
			sendTime))

	if !msg.GetIsOnlySendTarget() {
		if err = l.Push2Myself(c, fatherSpan, msg.GetFromId(), resp); err != nil {
			return err
		}
	}

	return l.Push2Target(c, fatherSpan, msg.GetFromId(), msg.GetTargetId(), rawdata, resp)
}

// push to myself
func (l *Logic) Push2Myself(c context.Context, fatherSpan opentracing.Span, fromId string, resp *pb.ClientPackage) (err error) {
	// 如果玩家不在线
	data, err := l.dao.GetPlayerMapping(fromId)
	if err != nil {
		return nil
	}
	cometId := strings.Split(data, ",")[0]

	// push to nats
	if err = l.dao.PushMsg(c, fatherSpan, cometId, fromId, resp); err != nil {
		log.L().Warnf("logic Push2Myself, call PushMsg err: %v, fromId:%s", err, fromId)
		return
	}
	return
}

// push to target
func (l *Logic) Push2Target(c context.Context, fatherSpan opentracing.Span, from, target string, rawInfo *pb.ChatMsgData, resp *pb.ClientPackage) (err error) {
	// 如果玩家不在线
	data, err := l.dao.GetPlayerMapping(target)
	if err != nil {
		// save to redis
		err = l.dao.SavePersonalMsg(from, target, rawInfo)
		if err != nil {
			return
		}
		return nil
	}
	cometId := strings.Split(data, ",")[0]

	// push to nats
	if err = l.dao.PushMsg(c, fatherSpan, cometId, target, resp); err != nil {
		log.L().Warnf("logic Push2Target, call PushMsg err: %v, targetId:%s", err, target)
		return
	}
	return
}

// PushRoom push a message by room.
func (l *Logic) PushRoom(c context.Context, fatherSpan opentracing.Span, msg *pb.GamexChatMsg, sendTime int64) (err error) {
	log.L().Infof("push room:%s, from:%s, msg: %s, bubble: %d", msg.GetFromId(), msg.GetTargetId(), msg.GetContent(), msg.GetBubble())

	// raw data
	info := &pb.ChatMsgData{
		Uuid:           proto.String(uuid.NewV4().String()),
		FromId:         msg.FromId,
		TargetId:       msg.TargetId,
		Type:           pb.ChatType_Room.Enum(),
		Content:        msg.Content,
		SdkParam:       msg.SdkParam,
		SendStamp:      &sendTime,
		IsTrumpet:      msg.IsTrumpet,
		IsGmMarquee:    msg.IsGmMarquee,
		IsGameMarquee:  msg.IsGameMarquee,
		TemplateInfo:   msg.TemplateInfo,
		GmMarquee:      msg.GmMarquee,
		ExtraParam:     msg.ExtraParam,
		TrumtType:      msg.TrumtType,
		VoiceChatUrl:   msg.VoiceChatUrl,
		NeedCross2Zone: msg.NeedCross2Zone,
		Bubble:         msg.Bubble,
	}
	r := &pb.ChatMsgResp{
		Msg:  info,
		Code: pb.ErrorCode_Success.Enum(),
	}

	resp, err := generateChatMsgResp(r)
	if err != nil {
		log.L().Warnf("logic push room, generateChatMsgResp failed, fromId:%s roomId:%s, conetent:%s", msg.GetFromId(), msg.GetTargetId(), msg.GetContent())
		return
	}

	// save room msg to redis
	err = l.dao.SaveRoomMsg(msg.GetTargetId(), info)
	if err != nil {
		return
	}

	// platform log
	chatlog.LogChatPlatform(
		logiclog.CommonInfo(
			msg.GetFromId(), // operator
			chatlog.RoomChat,
			true,               // 是否是房间消息
			msg.GetIsTrumpet(), // 是否是喇叭消息
			msg.GetFromId(),    // sender
			msg.GetExtraParam().GetPlayerShowInfo().GetName(),
			msg.GetTargetId(),
			"",               // device
			msg.GetContent(), // 文本内容
			sendTime))

	return l.dao.BroadcastRoomMsg(c, fatherSpan, msg.GetTargetId(), resp)
}

// get history msg
func (l *Logic) GetPersonHistory(c context.Context, fromId, targetId string, begin, end uint32) error {
	log.L().Debugf("get history targetId:%s, begin:%d, end: %d", targetId, begin, end)

	data, err := l.dao.GetPlayerMapping(fromId)
	if err != nil {
		return errors.New(pb.ErrorCode_RedisTargetDataError.String())
	}
	cometId := strings.Split(data, ",")[0]

	// get save msg from redis
	his, err := l.dao.FetchPersonMsg(targetId, 0, uint32(config.Cfg.Personal.MaxPersonSaveCount))
	if err != nil {
		return err
	}
	log.L().Debugf("GetPersonHistory FetchPersonMsg num: %v", len(his))

	needSendTimes := len(his) / PersonalHistoryPerTime
	for i := 0; i < needSendTimes; i++ {
		msg, err := generateHistoryMsgResp(his[i*PersonalHistoryPerTime:(i+1)*PersonalHistoryPerTime], pb.ErrorCode_Success)
		if err != nil {
			log.L().Warnf("History generate history msg resp failed, targetId:%s, begin:%d, end:%d", targetId, begin, end)
			return err
		}

		// send history to nats
		l.dao.PushHistoryMsg(c, cometId, fromId, msg)
		log.L().Debugf("GetPersonHistory PushHistoryMsg startIndex: %v, endIndex: %v", i*PersonalHistoryPerTime, (i+1)*PersonalHistoryPerTime)
	}

	if len(his)-needSendTimes*PersonalHistoryPerTime > 0 {
		msg, err := generateHistoryMsgResp(his[needSendTimes*PersonalHistoryPerTime:], pb.ErrorCode_Success)
		if err != nil {
			log.L().Warnf("History generate history msg resp failed, targetId:%s, begin:%d, end:%d", targetId, begin, end)
			return err
		}

		// send history to nats
		l.dao.PushHistoryMsg(c, cometId, fromId, msg)
		log.L().Debugf("GetPersonHistory PushHistoryMsg startIndex: %v, endIndex: %v", needSendTimes*PersonalHistoryPerTime, len(his))
	}

	return nil
}

func (l *Logic) GetRoomHistory(c context.Context, fromId, targetId string, begin, end uint32) error {
	log.L().Debugf("get room history targetId:%s, begin:%d, end: %d", targetId, begin, end)
	roomType := const_value.GetRoomType(targetId)
	roomSaveCount := config.GetRoomSaveHistoryMsgCount(roomType)

	var his []*pb.ChatMsgData
	if roomSaveCount > 0 {
		if begin > end {
			return errors.New(pb.ErrorCode_HistoryArgIndexError.String())
		}
		if begin < 0 || end < 0 {
			return errors.New(pb.ErrorCode_HistoryArgIndexError.String())
		}
		if begin > uint32(roomSaveCount) {
			return errors.New(pb.ErrorCode_HistoryArgIndexError.String())
		}
		if end > uint32(roomSaveCount) {
			end = uint32(roomSaveCount)
		}

		// get save msg from redis
		var err error
		his, err = l.dao.FetchRoomMsg(targetId, begin, end)
		if err != nil {
			return err
		}
	}

	msg, err := generateHistoryMsgResp(his, pb.ErrorCode_Success)
	if err != nil {
		log.L().Warnf("room History generate history msg resp failed, targetId:%s, begin:%d, end:%d", targetId, begin, end)
		return err
	}

	data, err := l.dao.GetPlayerMapping(fromId)
	if err != nil {
		return errors.New(pb.ErrorCode_RedisTargetDataError.String())
	}
	cometId := strings.Split(data, ",")[0]

	// send history to nats
	return l.dao.PushHistoryMsg(c, cometId, fromId, msg)
}

func (l *Logic) PushForbidden(c context.Context, targetId, reason string, endTime int64) (err error) {
	var (
		resp = &pb.ClientPackage{}
		buf  []byte
	)
	if targetId == "" || reason == "" || endTime == 0 {
		return errors.New("arg is invalid")
	}
	log.L().Debugf("logic PushForbidden targetId:%s, reason:%s endTime:%d", targetId, reason, endTime)
	data, err := l.dao.GetPlayerMapping(targetId)
	// 如果不在线直接保存数据 不发送到comet
	if err != nil {
		/*
			// save forbidden to redis
			err = l.dao.SetPlayerForbidden(targetId, reason, endTime)
			// 如果保存还失败了 需要返回失败
			if err != nil {
				log.L().Warnf("player not online, push forbidden save info to redis failed, target is:%s, reason:%s, endTime:%d", targetId, reason, endTime)
				return errors.New(pb.ErrorCode_RedisTargetDataError.String())
			}
		*/
		// 玩家不在线 保存成功
		return nil
	}
	cometId := strings.Split(data, ",")[0]

	r := &pb.ForbiddenResp{
		Reason:  &reason,
		EndTime: &endTime,
	}

	buf, err = proto.Marshal(r)
	if err != nil {
		return
	}

	resp.RawData = buf
	resp.MessageId = proto.Uint32(uint32(pb.ChatOperation_S2CForbidden))

	/*
		// save forbidden to redis
		if err = l.dao.SetPlayerForbidden(targetId, reason, endTime); err != nil {
			log.L().Warnf("push forbidden save info to redis failed, target is:%s, reason:%s, endTime:%d", targetId, reason, endTime)
			return
		}
	*/

	// push to nats
	if err = l.dao.PushMsg(c, nil, cometId, targetId, resp); err != nil {
		log.L().Warnf("logic PushKey, call PushMsg err: %v, targetId:%s,", err, targetId)
		return
	}
	return
}

func (l *Logic) PushUnForbidden(c context.Context, targetId string) (err error) {
	var (
		resp = &pb.ClientPackage{}
	)
	if targetId == "" {
		return errors.New("target is empty")
	}

	log.L().Debugf("logic Push UnForbidden targetId:%s", targetId)
	data, err := l.dao.GetPlayerMapping(targetId)
	if err != nil {
		/*
			saveErr := l.dao.DelPlayerForbidden(targetId)
			if saveErr != nil {
				return errors.New(pb.ErrorCode_RedisTargetDataError.String())
			}
		*/
		return nil
	}
	cometId := strings.Split(data, ",")[0]

	resp.RawData = nil
	resp.MessageId = proto.Uint32(uint32(pb.ChatOperation_S2CUnForbidden))

	// l.dao.DelPlayerForbidden(targetId)
	// push to nats
	if err = l.dao.PushMsg(c, nil, cometId, targetId, resp); err != nil {
		log.L().Warnf("logic PushKey, call PushMsg err: %v, targetId:%s,", err, targetId)
		return
	}
	return
}

// PushAll push a message to all.
func (l *Logic) PushAll(c context.Context, op, speed int32, msg []byte) (err error) {
	log.L().Infof("push all")
	// return l.dao.BroadcastMsg(c, op, speed, msg)
	return
}

func (l *Logic) RoomDestroy(c context.Context, rooms []string) (err error) {
	return l.dao.RoomDestroy(rooms)
}

func (l *Logic) PushTranslate(c context.Context, msg *pb.TranslateWithTarget) (err error) {
	var (
		cnt    int64
		result string
	)
	targetId := msg.GetTarget()
	data, err := l.dao.GetPlayerMapping(targetId)
	if err != nil || len(data) == 0 {
		return nil
	}
	cometId := strings.Split(data, ",")[0]

	now := time.Now()
	k4stat := "translate"
	name := fmt.Sprintf("%s.%s.%s", fmt.Sprint(conf.Cfg.Gid), fmt.Sprint(conf.Cfg.ServerId), k4stat)
	cnt, result, err = translate.Translate(msg.GetReq().GetTarget(), msg.GetReq().GetText())
	metrics.TimeStatisticsWithDelayTime(name, k4stat, time.Now().Sub(now).Nanoseconds())
	if err != nil {
		log.L().Warnf("translate fail: %v %v", msg, err)
	}

	// translate statistics
	chatlog.LogTranslate(&chatlog.TranslateInfo{
		Gid:              config.Cfg.Gid,
		LogicId:          config.Cfg.ServerId,
		Sender:           msg.GetTarget(),
		TargetLanguage:   msg.GetReq().GetTarget(),
		Content:          msg.GetReq().GetText(),
		TranslateContent: result,
		TranslateNum:     cnt,
	})

	// 翻译成功、失败都返回
	msgUuid := msg.GetReq().GetUuid()
	index := msg.GetReq().GetIndex()
	rsp := &pb.TranslateResp{
		Uuid:   &msgUuid,
		Result: &result,
		Index:  &index,
	}

	buf, _ := proto.Marshal(rsp)
	r := &pb.ClientPackage{
		MessageId: proto.Uint32(uint32(pb.ChatOperation_S2CTranslate)),
		RawData:   buf,
	}
	if err = l.dao.PushMsg(c, nil, cometId, targetId, r); err != nil {
		log.L().Warnf("logic call push translate err: %v, targetId:%s,", err, targetId)
		return
	}
	return
}
