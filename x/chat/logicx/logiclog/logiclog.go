package logiclog

import (
	"strings"

	"github.com/nghichtu91/platform/share/planx/tilogs/chatlog"
	"github.com/nghichtu91/platform/share/x/chat/logicx/config"
)

func CommonInfo(operator, typ string, isroom, istrumpet bool, fromId, senderName, targetId, device, content string, sendtime int64) *chatlog.ChatPlatFormInfo {
	commonInfo := &chatlog.ChatPlatFormInfo{}
	commonInfo.LogicId = config.Cfg.ServerId
	commonInfo.Gid = config.Cfg.Gid
	commonInfo.Operator = operator
	commonInfo.Type = typ
	commonInfo.IsRoomMsg = isroom
	commonInfo.Sender = fromId
	commonInfo.Receiver = targetId
	commonInfo.SenderDevcie = device
	commonInfo.Content = content
	commonInfo.SendTime = sendtime
	commonInfo.SenderName = senderName
	commonInfo.IsTrumpet = istrumpet

	splitSlice := strings.Split(operator, ":")
	if len(splitSlice) != 3 {
		commonInfo.GameSerId = ""
		return commonInfo
	}

	commonInfo.GameSerId = splitSlice[1]
	return commonInfo
}
