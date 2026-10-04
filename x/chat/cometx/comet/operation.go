package comet

import (
	"context"
	"errors"

	// "sync/atomic"
	// "time"

	"github.com/golang/protobuf/proto"
	log "github.com/nghichtu91/platform/share/planx/tilogs"

	// "github.com/nghichtu91/platform/share/x/chat/api/const_value"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
	"github.com/nghichtu91/platform/share/x/chat/cometx/comet/resetresp"
	"github.com/nghichtu91/platform/share/x/chat/cometx/config"
)

// Connect connected a connection.
func (s *Server) Connect(c context.Context, acid, token, sessionId string) (key, reason string, endTime int64, err error) {
	var reply *pb.ConnectReply
	rpcClient := s.RPCClient()
	if rpcClient == nil {
		err = errors.New(pb.ErrorCode_LogicRpcNotAvailable.String())
		return
	}
	reply, err = rpcClient.Connect(c, &pb.ConnectReq{
		Server:    &config.Cfg.ServerId,
		Acid:      &acid,
		Token:     &token,
		SessionId: &sessionId,
	})
	if err != nil {
		return
	}

	return reply.GetAcid(), reply.GetReason(), reply.GetEndTime(), nil
}

func (s *Server) SendChatMsg(c context.Context, p *pb.ClientPackage, session *Session) (err error) {
	/*
		var (
			resp    = &pb.ChatMsgResp{}
			msg     = &pb.ChatMsg{}
			errInfo *pb.ErrorInfo
		)
		// 重用req包 然后直接返回
		defer func() {
			resetresp.ResetChatResp(p, resp, err)
		}()

		if s.rpcClient == nil {
			err = errors.New(pb.ErrorCode_LogicRpcNotAvailable.String())
			return
		}

		// 发给logic的数据还需要一个from id
		err = proto.Unmarshal(p.RawData, msg)
		if err != nil {
			return
		}
		// 不打内容了 太大了
		log.L().Infof("SendChatMsg key:%s", msg, session.Key)

		chatReq := &pb.ChatMsgReq{
			FromId:   &session.Key, ClienReq: msg,
		}

		// 禁言中
		if atomic.LoadInt64(&session.ForbiddenEndTime) >= time.Now().Unix() ||
			atomic.LoadInt64(&session.ForbiddenEndTime) == int64(const_value.ForbiddenForever) {
			return errors.New(pb.ErrorCode_ForbiddenChat.String())
		}

		if msg.GetType() == pb.ChatType_Room {
			// 1.玩家是否在这个房间内
			if !session.HasRoom(msg.GetTargetId()) {
				return errors.New(pb.ErrorCode_RoomDrop.String())
			}
			// 2.玩家是否可以往这个房间发送消息(比如需要消耗喇叭的消息 不能让客户端直接发送消息)
			if isForbiddenRoom(msg.GetTargetId()) {
				return errors.New(pb.ErrorCode_RoomForbidden.String())
			}
			// 3.防止刷屏房间
			if session.InRoomCd(msg.GetTargetId()) {
				return errors.New(pb.ErrorCode_RoomCd.String())
			}
			session.RoomCd(msg.GetTargetId())
		}

		if errInfo, err = s.rpcClient.SendChatMsg(c, chatReq); err != nil {
			return
		}
		if errInfo.Err == nil {
			return nil
		}
		return errors.New(errInfo.GetErr())
	*/
	return nil
}

func (s *Server) JoinRoom(c context.Context, p *pb.ClientPackage, session *Session, b *Bucket) (err error) {
	// return to client resp
	resp := &pb.JoinRoomResp{}
	defer func() {
		resetresp.ResetJoinRoomResp(p, resp, err)
	}()

	msg := &pb.JoinRoom{}
	err = proto.Unmarshal(p.RawData, msg)
	if err != nil {
		log.L().Errorf("JoinRoom Unmarshal err %v", err)
		return
	}
	log.L().Debugf("JoinRoom msg:%v, s:%s", msg, session.Key)

	for _, v := range msg.RoomId {
		if err = b.JoinRoom(v, session); err != nil {
			return
		}
		resp.RoomId = append(resp.RoomId, v)
	}
	return
}

func (s *Server) LeaveRoom(c context.Context, p *pb.ClientPackage, session *Session, b *Bucket) (err error) {
	resp := &pb.LeaveRoomResp{}
	delroom := false
	defer func() {
		resetresp.ResetLeaveRoomResp(p, resp, err)
	}()

	msg := &pb.LeaveRoom{}
	err = proto.Unmarshal(p.RawData, msg)
	if err != nil {
		log.L().Errorf("LeaveRoom Unmarshal err %v", err)
		return
	}
	log.L().Debugf("LeaveRoom msg:%v, s:%s", msg, session.Key)

	// 记录所有需要销毁的room
	needDestroyRooms := make([]string, 0)
	for _, v := range msg.RoomId {
		err, delroom = b.LeaveRoom(v, session)
		if err != nil {
			log.L().Errorf("LeaveRoom b.LeaveRoom err %v", err)
			return
		}

		if delroom {
			needDestroyRooms = append(needDestroyRooms, v)
		}
		resp.RoomId = append(resp.RoomId, v)
	}

	if len(needDestroyRooms) > 0 {
		rpcClient := s.RPCClient()
		if rpcClient == nil {
			log.L().Errorf("Leave room, need destroy room, but rpcClient is nil!")
			return errors.New(pb.ErrorCode_LogicRpcNotAvailable.String())
		}

		if _, err = rpcClient.RoomDestroy(context.Background(), &pb.RoomDestroyReq{
			RoomIds: needDestroyRooms,
		}); err != nil {
			log.L().Errorf("Leave room, need destroy room, but failed! err:%v", err)
		}
	}
	return
}

// GetHistoryMsg 将客户端请求历史消息转发到logicx
func (s *Server) GetHistoryMsg(c context.Context, p *pb.ClientPackage, session *Session) (err error) {
	var (
		resp    = &pb.HistoryMsgResp{}
		msg     = &pb.HistoryMsg{}
		errInfo = &pb.ErrorInfo{}
	)

	defer func() {
		resetresp.ResetHistoryResp(p, resp, err)
	}()

	// 修复线上cometx panic
	if p == nil || p.RawData == nil {
		log.L().Errorf("get history package :%v, raw data is nil", p)
		return errors.New(pb.ErrorCode_ProtoMarshalFailed.String())
	}

	proto.Unmarshal(p.RawData, msg)
	// 发给logic的数据还需要一个from id
	hisReq := &pb.HistoryMsgReq{
		FromId:   &session.Key,
		ClienReq: msg,
	}

	log.L().Debugf("get history msg, msg:%v, s:%s", msg, session.Key)

	rpcClient := s.RPCClient()
	if rpcClient == nil {
		log.L().Errorf("GetHistoryMsg rpcClient nil")
		return errors.New(pb.ErrorCode_LogicRpcNotAvailable.String())
	}

	// 如果是私聊 那么只能拉取自己的消息
	if msg.GetType() == pb.ChatType_Personal {
		if msg.GetTargetId() != session.Key {
			log.L().Errorf("GetHistoryMsg ChatType_Personal err %s", pb.ErrorCode_HistoryArgTargetError.String())
			return errors.New(pb.ErrorCode_HistoryArgTargetError.String())
		}
	} else if msg.GetType() == pb.ChatType_Room {
		// 如果是房间消息 看玩家是否在这个房间
		if !session.HasRoom(msg.GetTargetId()) {
			log.L().Errorf("GetHistoryMsg ChatType_Room err %s", pb.ErrorCode_RequestOperationNotValid.String())
			return errors.New(pb.ErrorCode_RequestOperationNotValid.String())
		}
	}

	if errInfo, err = rpcClient.GetHistoryMsg(c, hisReq); err != nil {
		return
	}
	if errInfo.Err == nil {
		return nil
	}
	log.L().Errorf("GetHistoryMsg err %s", errInfo.GetErr())
	return errors.New(errInfo.GetErr())
}

func (s *Server) Translate(c context.Context, p *pb.ClientPackage, session *Session) (err error) {
	var (
		req     = &pb.TranslateReq{}
		errInfo = &pb.ErrorInfo{}
	)

	_ = proto.Unmarshal(p.RawData, req)
	log.L().Debugf("get translate msg, msg:%v, s:%s", req, session.Key)

	rpcClient := s.RPCClient()
	if rpcClient == nil {
		log.L().Errorf("Translate rpcClient nil")
		return errors.New(pb.ErrorCode_LogicRpcNotAvailable.String())
	}

	msg := &pb.TranslateWithTarget{
		Target: &session.Key,
		Req:    req,
	}
	if errInfo, err = rpcClient.Translate(c, msg); err != nil {
		return
	}
	if errInfo.Err == nil {
		return nil
	}
	log.L().Errorf("Translate rpcClient.Translate err:%v", errInfo.GetErr())
	return errors.New(errInfo.GetErr())
}

// Disconnect disconnected a connection.
func (s *Server) Disconnect(c context.Context, key, sessionId string) (err error) {
	rpcClient := s.RPCClient()
	if rpcClient == nil {
		return errors.New(pb.ErrorCode_LogicRpcNotAvailable.String())
	}
	_, err = rpcClient.DisConnect(context.Background(), &pb.DisconnectReq{
		Server:    &s.serverID,
		Key:       &key,
		SessionId: &sessionId,
	})
	return nil
}

// Heartbeat heartbeat a connection session.
func (s *Server) Heartbeat(ctx context.Context, key string) (err error) {
	rpcClient := s.RPCClient()
	if rpcClient == nil {
		return errors.New(pb.ErrorCode_LogicRpcNotAvailable.String())
	}

	_, err = rpcClient.Heartbeat(ctx, &pb.HeartbeatReq{
		Server: &s.serverID,
		Key:    &key,
	})
	return
}

// Operate comet会处理一部分操作 比如房间类的操作 剩下的转发给logic
// go 更常使用的是error的错误处理 与客户端最好还是使用错误码的方式处理 也方便多语言之类的
func (s *Server) Operate(ctx context.Context, p *pb.ClientPackage, session *Session, b *Bucket) bool {
	notify := true
	switch pb.ChatOperation(p.GetMessageId()) {
	case pb.ChatOperation_C2SJoinRoom:
		s.JoinRoom(ctx, p, session, b)
	case pb.ChatOperation_C2SLeaveRoom:
		s.LeaveRoom(ctx, p, session, b)
	case pb.ChatOperation_C2SChatMsg:
		err := s.SendChatMsg(ctx, p, session)
		// 如果在comet已经有错误返回 直接返回包
		notify = err != nil
	case pb.ChatOperation_C2SHistroyMsg:
		// 如果在comet已经有错误返回 直接返回包
		err := s.GetHistoryMsg(ctx, p, session)
		notify = err != nil
	case pb.ChatOperation_C2STranslate:
		// 如果在comet已经有错误返回 直接返回包
		err := s.Translate(ctx, p, session)
		notify = err != nil
	default:
		p.RawData = p.RawData[:0]
		log.L().Errorf("opeate type error, messageId:%d, session id:%s", p.MessageId, session.Key)
	}
	return notify
}
