package dao

import (
	"context"
	"strconv"

	"github.com/opentracing/opentracing-go"

	"github.com/nghichtu91/platform/share/planx/nats_cli"
	log "github.com/nghichtu91/platform/share/planx/tilogs"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
	"github.com/nghichtu91/platform/share/x/chat/logicx/config"
)

// PushMsg push a message to databus.
func (dao *Dao) PushMsg(c context.Context, fatherSpan opentracing.Span, server string, key string, msg *pb.ClientPackage) error {
	var (
		err error
	)
	pushMsg := &pb.PushMsg{
		Type:   pb.PushMsg_PUSH.Enum(),
		Server: &server,
		Key:    &key,
		Msg:    msg,
	}

	gid := strconv.Itoa(int(config.Cfg.Gid))
	if fatherSpan != nil {
		err = nats_cli.SendMsgWithCtx(fatherSpan, nats_cli.ChatSubj(gid), pushMsg)
	} else {
		err = nats_cli.SendMsg(nats_cli.ChatSubj(gid), pushMsg)
	}
	if err != nil {
		log.L().Warnf("send msg to nats failed msg is: %v, err is %v", pushMsg, err)
		return err
	}

	log.L().Debugf("nats send personal msg successful server is %s,key is %s, messageId:%d", server, key, msg.MessageId)
	return nil
}

func (dao *Dao) BroadcastRoomMsg(c context.Context, fatherSpan opentracing.Span, roomId string, msg *pb.ClientPackage) (err error) {
	pushMsg := &pb.PushMsg{
		Type: pb.PushMsg_ROOM.Enum(),
		Room: &roomId,
		Msg:  msg,
	}

	gid := strconv.Itoa(int(config.Cfg.Gid))
	if fatherSpan != nil {
		err = nats_cli.SendMsgWithCtx(fatherSpan, nats_cli.ChatSubj(gid), pushMsg)
	} else {
		err = nats_cli.SendMsg(nats_cli.ChatSubj(gid), pushMsg)
	}
	if err != nil {
		log.L().Warnf("send room msg to nats failed msg is: %v, err is %v", pushMsg, err)
		return
	}

	log.L().Debugf("nats send personal msg successful roomId is %s, messageId:%d", roomId, msg.MessageId)
	return nil
}

// push history to nats
func (dao *Dao) PushHistoryMsg(c context.Context, cometId, targetId string, msg *pb.ClientPackage) error {
	var (
		err error
	)
	pushMsg := &pb.PushMsg{
		Type:   pb.PushMsg_HISTORY.Enum(),
		Server: &cometId,
		Key:    &targetId,
		Msg:    msg,
	}

	gid := strconv.Itoa(int(config.Cfg.Gid))
	err = nats_cli.SendMsg(nats_cli.ChatSubj(gid), pushMsg)
	if err != nil {
		log.L().Warnf("send history msg to nats failed,cometId:%s, targetId:%s, msg is: %v, err is %v", cometId, targetId, pushMsg, err)
		return err
	}

	log.L().Debugf("nats send history msg successful cometId is %s, targetId:%s, messageId:%d", cometId, targetId, msg.MessageId)
	return nil
}
