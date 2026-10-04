package config

import (
	"fmt"
	"time"
)

const (
	LogFileName = "log.toml"
)

const (
	LogService = "service"
	LogSubCmd  = "subcmd"
	Gift       = "gift"
	AllInOne   = "allinone"
)

/**
时间控制或显示配置相关。
*/
const Millisecond = 1000000                     // 纳秒与毫秒的转换[用于时间显示]。
const DefaultReqTimeOut = 2 * time.Second       // 默认的请求处理超时时间。
const TimeConsumeReqTimeOut = 600 * time.Second // 高耗时独立处理超时时间。

/**
子模块配置相关。
*/
const DefaultReqHandlerNum = 10      // 模块Hash协程处理请求时的默认协程数量（正常线程数目）。
const DefaultReqHandlerSmallNum = 5  // 模块Hash协程处理请求时的默认协程数量（少量线程数目）。
const DefaultReqHandlerSingleNum = 1 // 模块Hash协程处理请求时的默认协程数量（单一线程数目）。
const DefaultChanBufferNum = 2048    // Channel的默认缓冲数量（正常缓冲数目）。
const DefaultChanBufferSmallNum = 10 // Channel的默认缓冲数量（极少缓冲数目）。

/**
兑换码生成配置相关。
*/
const (
	// 兑换码组成的各部分上限。
	CodeMaxGenNum   = 10000000 // 一个批次组最大生成兑换码数量：100万（0~999.9999万）。
	CodeMaxBatchNum = 100000   // 一个项目最大准许批次：10万（0~9.9999万）。
	CodeMaxGroupNum = 100      // 一个批次最大准许组：100（0~99）。
	CodeMaxProject  = 100      // 最大准许项目：99（0~99）。

	// 兑换码生成的各部分占位。
	CodeInherentCoding = 10       // 数字串-固定值：1位。
	CodeProjectCoding  = 100      // 数字串-项目代号：2位。
	CodeIndexCoding    = 10000000 // 数字串-生成序号：7位。
	CodeRandomCoding   = 10       // 数字串-随机序号：1位。
	CodeBatchCoding    = 100000   // 数字串-批次号：5位。
	CodeGroupCoding    = 100      // 数字串-组号：2位。

	// 控制特殊处理的常量数值。
	CodeInherent          = 1                // 兑换码数字串最高位固定值（为了保证生成的字符串为14位定长）。
	CodeRandomCommonRange = CodeRandomCoding // 随机的最大范围[0,RandomRange).
	CodeRandomFirstRange  = 8                // 首位随机的数值[0,7)（int64最高位184，为了防止溢出，这里最高为7）.
	CodeFirstGeneration   = 0                // 目标批次和组号首次生成时，已拥有的兑换码数量为0.
)

/**
兑换码类别。
*/
const (
	// 普通兑换码:同一批次下，一个玩家只能用一个。每个码仅可一人使用一次。
	TypeGiftCodeNormal = 1
	// 通用兑换码:同一批次组号下仅有一个码，不设置使用上限，每人可使用一次。
	// 可设置为自定义兑换码（需要包含数字）。
	TypeGiftCodeUniversal = 2
	// 多用兑换码:同一批次组号下仅有一个码，设置使用上限，每人可使用一次。
	// 可设置为自定义兑换码（需要包含数字）。
	TypeGiftCodeMulti = 3
)

/**
正则匹配规则
*/
const (
	RegexpContainNum = "[0-9]" // 字符串包含数字字符。
)

/**
数字码与字符码转换相关。
*/
// 26个字母的乱序大写集合，对应数字（0~25）。
var Num2Char = "HAUEQWVF" + "GRLKTNPZ" + "MBCXIJOY" + "DS"

// 26个字母的对应数字反查:Map[Base26Char]Num
var Char2Num = map[string]uint64{
	"H": 0, "A": 1, "U": 2, "E": 3, "Q": 4, "W": 5, "V": 6, "F": 7,
	"G": 8, "R": 9, "L": 10, "K": 11, "T": 12, "N": 13, "P": 14, "Z": 15,
	"M": 16, "B": 17, "C": 18, "X": 19, "I": 20, "J": 21, "O": 22, "Y": 23,
	"D": 24, "S": 25,
}

/**
数据库、表相关。
*/
const (
	// 注意！表名改动会导致之前的内容无法被链接！请不要随意改动！
	TableGidVisibility  = "GidVisibilityGiftCode" // 大区为主键，批次号和分组号为结构体构建的切片为内容。（数量较少不做拆分）
	TableCustomCode     = "CustomGiftCode"        // 自定义兑换码为主键，批次号、分组号、使用者为内容。（数量较少不做拆分）
	TableGenCode        = "GenGiftCode"           // 生成兑换码为主键，使用者为内容（批次号、分组号可以从兑换码中解出）。（需要拆分）
	TableGiftCodeConfig = "GiftCodeConfig"        // 批次号、分组号为主键，配置（统称）和兑换码集合为内容。（需要拆分）
	TableL2AGid         = "Logic2ActualGid"       // 逻辑大区（隔离大区）到实际大区的映射关系表。
	TableA2LGid         = "Actual2LogicGid"       // 实际大区到逻辑大区（隔离大区）的映射关系表。
	TableGlobalInfo     = "GlobalInfo"            // 全局信息表。

	// 注意！文件名改动会导致之前的内容无法被链接！请不要随意改动！
	FileCodesNamePrefix = "Codes"             // 在OSS平台存储的码集文件的前缀。
	PathCodesName       = "gift_server_codes" // 在OSS平台存储的码集文件的路径。

	// 注意！拆分数改动会导致之前的内容链接错位！请不要随意改动！

	/**
	分析一：国服普通码使用较多，海外通用码使用较多。（10份划分）国内普：通=5：5，海外普：通=1：9。
	分析二：生成兑换码以10个批次作为划分，单表上限行10（批次）*100（组）*10000000（码）100亿。
	分析三：通常情况下组号不会超过10个。码量在百万级别。

	结论：通常情况下，以10个批次作为兑换码表的划分，国服表行数为5*10*100万，5000万行，海外表行数为1*10*100万，1000万行。
	*/
	GenCodeSplitByBatch = 10 // 按照指定的批次数量拆分 TableGenCode 表。

	/**
	分析一：兑换码配置表以10000个批次作为划分，单表上限行10000（批次）*100（组）100万。
	分析二：通常情况下分组在10个左右，通常单表行10万行。

	结论：通常情况下，以100个批次作为配置表的划分。
	*/
	GiftCodeConfigSplitByBatch = 10000 // 按照指定的批次数量拆分 TableGiftCodeConfig 表。

	InsertGenCodeBatchNum = 10000 // 当生成大量兑换码批量插入数据库时，一个插入批次包含的条目数。
)

const (
	FlagSplice = ", "

	ConfigFlagFull  = "config"
	ConfigFlagAbbre = "c"
	ConfigFlag      = ConfigFlagFull + FlagSplice + ConfigFlagAbbre

	LogicLogFlagFull  = "logiclog"
	LogicLogFlagAbbre = "ll"
	LogicLogFlag      = LogicLogFlagFull + FlagSplice + LogicLogFlagAbbre

	PortFlagFull  = "port"
	PortFlagAbbre = "p"
	PortFlag      = PortFlagFull + FlagSplice + PortFlagAbbre
)

// GetDBConnMySql 获取连接MySql数据库的命令语句
func GetDBConnMySql(url string) string {
	return fmt.Sprintf("%s?allowNativePasswords=true", url)
}
