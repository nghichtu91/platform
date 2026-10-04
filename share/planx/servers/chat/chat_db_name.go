package chat

import "fmt"

// 登录时 用于userId与token的映射关系
func GetTokenKey(userId string) string {
	return fmt.Sprintf("chat:%s:%s", Logining, userId)
}

// 登陆之后 用于userId与comet的映射关系
func GetMappingKey(userId string) string {
	return fmt.Sprintf("chat:%s:%s", Logined, userId)
}

// 私聊信息的key f为私聊发送人 t为私聊接受人
// 暂时废弃
func GetSaveKey(from, target string) string {
	return fmt.Sprintf("chat:f:%st:%s", from, target)
}

// 私聊信息的key 只存发给自己的离线消息
func GetPersonalKey(target string) string {
	return fmt.Sprintf("chat:personal:%s", target)
}

// 房间消息
func GetRoomSaveKey(roomId string) string {
	return fmt.Sprintf("chat:r:%s", roomId)
}

// 禁言key
func GetForbiddenKey(userId string) string {
	return fmt.Sprintf("chat:forbidden:%s", userId)
}
