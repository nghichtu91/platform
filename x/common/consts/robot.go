package consts

const (
	RobotSuffix         = "@robot.robot"           // 机器人登录token后缀以及flag
	RobotDevice         = "Optimus Prime"          // 机器人登录时device字段
	RobotDeviceIDSuffix = "@robot@robot"           // 机器人deviceID后缀
	RobotNamePrefix     = "stress"                 // 机器人名字前缀
	FakeShardID         = "999999"                 // 用于gate模拟握手的shardID
	FakeShardIDInt      = 65535                    // battle服生成GRoomID使用
	RobotCometxID       = "robotID@cometx"         // cometx模拟握手ID
	RobotCometxToken    = "robotToken@cometx"      // cometx模拟握手Token
	RobotCometxRoom     = "robotChannel_invisible" // cometx特殊房间，应当只有server check使用
	MaxRobotChatPktSize = 10240                    // 最大聊天包长度
	Robot9v9NamePrefix  = "stress9v9_"             // 机器人三队演武名字前缀 后面拼上玩家名次
)
