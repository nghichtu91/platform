package errs

const Success = ""
const InvalidCodesJson = "" // 无效的代码集合。
const IDNotGen = -1         // ID尚未生成。

// 架构设计层
const (
	// services层异常。
	ErrUnknownApiType       = "未知的API请求类型"
	ErrBindReqData          = "绑定请求数据失败"
	ErrExchangeReqInterface = "转换请求接口失败"

	// module层Handle异常。
	ErrRegButNotAcceptance = "注册但是未提供受理函数"
	ErrHashChannelFull     = "Hash目标的Channel已满"
	ErrHashChannelTimeout  = "处理协程处理超时"
)

// 通用处理函数部分。
const (
	ErrCodeOverLimit         = "兑换码组成部分超出上限"
	ErrCodeEmptyCharBase     = "数字码到字符码的映射为空"
	ErrCodeCanNotDistinguish = "无法识别兑换码中的字符"
)

// 请求处理模块
const (
	// 通用
	ErrHandlerJsonUnMarshal            = "接受到GmTools的请求后，JsonUnmarshal异常"
	ErrHandlerBatchGroupIDHasInUsed    = "此批次和组已被使用"
	ErrHandlerBatchGroupIDHasNotInUsed = "此批次和组还未生成"
	ErrHandlerCustomCodeNotExist       = "自定义兑换码不存在"

	// 生成
	ErrHandlerNormalCodeNotCustomCode  = "普通兑换码不允许为自定义兑换码"
	ErrHandlerCustomCodeMustContainNum = "自定义兑换码必须包含数字作为组成部分"
	ErrHandlerAtLeastOneGidVisibility  = "生成的兑换码至少对一个大区是可见的"
	ErrHandlerDuplicateCode            = "重复的自定义礼包兑换码"

	// 追加
	ErrHandlerOnlyNormalAllowAppend = "只有普通兑换码允许追加"

	// 查询
	ErrHandlerTargetNotExist = "目标信息不存在"
	ErrHandlerCannotFindInfo = "无法查询到目标信息"

	// 销毁
	ErrHandlerDestroyContent              = "要销毁的目标兑换码集合为空"
	ErrHandlerDestroyCodesNotMatchRequest = "销毁的兑换码集合不满足设定要求"
	ErrHandlerDestroyCustomCodeOnlyOne    = "销毁自定义兑换码时数量仅可为一"
	ErrHandlerDestroyLimitNotExist        = "销毁的兑换码限定批次组不存在"
	ErrHandlerDestroyCustomCode           = "销毁自定义兑换码时异常"
	ErrHandlerDestroyGenCode              = "销毁生成兑换码时异常"
	ErrHandlerDestroyRepeated             = "销毁生成兑换码集合包含重复码"

	// 修改
	ErrHandlerUpdateNotAllowChangeType   = "礼包码的类型不允许被修改"
	ErrHandlerUpdateNotAllowChangeCustom = "自定义礼包码不允许修改"

	// 历史
	ErrHandlerQueryGenInfoRequestGid = "查询历史信息时至少给出一个Gid"

	// 领取
	ErrHandlerPlayerHasUseThisBatch = "玩家已经使用了此批次"
	ErrHandlerFindUseInfo           = "获取兑换码使用信息时失败"
	ErrHandlerCodeHasBeenDestroy    = "兑换码已经被销毁"
	ErrHandlerCodeHasReachUseLimit  = "兑换码已达使用上限"
	ErrHandlerCodeNotInValidTime    = "兑换码不在领取有效期内"
	ErrHandlerChannelNotMatch       = "玩家所在渠道不匹配"
	ErrHandlerGidNotMatch           = "玩家所在大区不匹配"
)

// DB模块
const (
	// 获取结果后读取信息。
	ErrDBScan = "读取行时出现异常"
	ErrDBData = "读取出的数据异常"

	// Json
	ErrDBJsonUnmarshal = "Json数据unmarshal时异常"
	ErrDBJsonMarshal   = "Json数据marshal时异常"

	// OSS
	ErrDBOSSUpload   = "向OSS数据库上传时异常"
	ErrDBOSSDownload = "从OSS数据库下载时异常"

	// OSS+Mysql
	ErrDBOSSUploadNoEffect = "虽然事务执行异常但是除OSS下载外不受影响，请勿重复操作"

	// 事务
	ErrDBTransactionBegin                      = "开启MySql事务异常"
	ErrDBCommitTransactionAndRollback          = "事务提交时失败,且无法回滚"
	ErrDBCommitTransaction                     = "事务提交时失败,已执行回滚"
	ErrDBDestroyCodesTransactionAndRollback    = "销毁兑换码事务执行异常且无法回滚"
	ErrDBOpGenCodeTablesTransactionAndRollback = "兑换码相关表操作事务执行异常且无法回滚"
	ErrDBInsertGenCodeTransactionAndRollback   = "生成兑换码表插入事务执行异常且无法回滚"

	// Sql
	ErrDBCloseSqlConn               = "关闭Sql连接时异常"
	ErrDBCheckAndCreateTable        = "数据表检查创建异常"
	ErrDBInsertGenCode              = "生成兑换码表插入执行异常"
	ErrDBQueryUseInfoByCodeAndBatch = "通过礼包码查询使用信息失败"
	ErrDBQueryUseInfoByBatchGroup   = "通过批次组查询使用信息失败"
	ErrDBBatchIDGroupIDNotInUsed    = "此批次和组尚未生成无法追加"
	ErrDBInsertCustomCode           = "自定义兑换码表插入执行异常"
	ErrDBQueryCustomCode            = "自定义兑换码表查询执行异常"
	ErrDBQueryGiftCodeConfig        = "获取兑换码配置异常"
	ErrDBUpdateGiftCodeConfig       = "更新兑换码配置异常"
	ErrDBInsertGiftCodeConfig       = "追加兑换码配置异常"
	ErrDBQueryGidVisibility         = "查询大区可见性异常"
	ErrDBUpdateGidVisibility        = "更新大区可见性异常"
	ErrDBInsertGidVisibility        = "插入大区可见性异常"
	ErrDBShowTablesWithPrefix       = "列举有指定前缀的表"
	ErrDBQueryGreatest              = "获取最大值信息异常"
	ErrDBUpdateGenUsesInfo          = "更新生成码信息异常"
	ErrDBUpdateCustomUsesInfo       = "更新自定义信息异常"
	ErrDBQueryGlobalGid             = "获取全大区配置异常"
	ErrDBUpdateGlobalGid            = "更新全大区配置异常"
)
