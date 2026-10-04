package common_log

/*
	字段名	参数	类型	描述
	:base
*/
const (
	base_event_id       = "event_id"       //事件ID,事件ID
	base_event_uuid     = "event_uuid"     //随机字符串,日志记录唯一标志码，防止记录重复生成
	base_event_time     = "event_time"     //时间戳,事件上传时间（秒，长整型）
	base_event_time2    = "event_time2"    //时间戳2,事件上传时间（日，如：2018-04-10）
	base_appkey         = "appkey"         //游戏标志,gameid，由英雄提供
	base_platform       = "platform"       //平台,IOS|ANDROID|WEB|SERVER|SERVER_TEST
	base_client_os      = "client_os"      //客户端类型,IOS|ANDROID|PAD|WEB
	base_server_id      = "server_id"      //区服id,使用英雄互娱统一服务器ID配置表
	base_channel_id     = "channel_id"     //渠道号（注册渠道）,使用英雄互娱统一渠道ID配置表
	base_app_channel_id = "app_channel_id" //次级渠道号（运营渠道）,使用英雄互娱统一渠道ID配置表
	base_device_id      = "device_id"      //设备号,设备号
	base_user_id        = "user_id"        //账号ID（聚合SDK关联ID）,聚合SDK关联ID
	base_open_id        = "open_id"        //渠道账号ID,渠道账号ID
	base_role_id        = "role_id"        //角色ID,角色ID
	base_role_key       = "role_key"       //角色唯一key,全服务器必须唯一，方便合服以及IPO操作
	base_transaction_id = "transaction_id" //事件关联ID,当一次事件触发多个日志产生时，需要由CP生成当前区服唯一ID用于关联用户的多个日志（同一行为触发的多个日志使用同一个关联ID），IPO必须 ,关联事件（获取代币，消耗代币，获取物品，消耗物品，商城购买，抽奖）
	base_user_ip        = "user_ip"        //用户IP,用户IP
	base_buess_time     = "buess_time"     //时间戳（秒，长整型）,业务发生时间（秒，长整型）
	base_time_zone      = "time_zone"      // 服务器所在时区

	Common_upload_bdc_device_id  = "bdc_device_id" // 新版本设备号
	Common_upload_device_id_type = "deviceid_type" // 设备号型号
	Common_upload_device_key     = "device_key"    // 设备号key
	Common_upload_device_mode    = "device_model"  // 设备型号
	Common_upload_ime            = "ime"           // 安卓设备IMEI标识符
	Common_upload_gaid           = "gaid"          // 安卓设备谷歌广告ID
	Common_upload_idfa           = "idfa"          // IOS设备广告标识符
	Common_upload_idfv           = "idfv"          // IOS设备供应商标识符
	Common_upload_android_id     = "android_id"    // 安卓ID
	Common_upload_oaid           = "oaid"          // 中广协OaId设备标识符
	Common_upload_sdk_version    = "sdk_version"   // BDCSdk版本号
	Common_upload_bdc_client_os  = "bdc_client_os" // 客户端类型
	Common_upload_old_device_id  = "old_device_id" // 历史上次设备号
)

/*
	日志说明：用户角色退出时输出
	操作类型：UserInfo
*/
const (
	UserInfo_event_id               = "10002"                  //事件id,10002
	UserInfo_role_name              = "role_name"              //角色名称,角色名称
	UserInfo_role_level             = "role_level"             //角色等级,角色等级
	UserInfo_vip_level              = "vip_level"              //VIP,VIP等级
	UserInfo_sex                    = "sex"                    //性别,性别 1男2女
	UserInfo_free_diamond_balance   = "free_diamond_balance"   //免费获取的一级代币剩余量,一级代币，免费获取的剩余量
	UserInfo_donate_diamond_balance = "donate_diamond_balance" //充值赠送的一级代币剩余量,一级代币，充值赠送的剩余量，如买1000赠送5%，首充双倍等。
	UserInfo_charge_diamond_balance = "charge_diamond_balance" //充值获得的一级代币剩余量,一级代币，充值获得的剩余量
	UserInfo_phy_balance            = "phy_balance"            //体力剩余量,体力剩余量
	UserInfo_month_card_balance     = "month_card_balance"     //月卡剩余领取次数,月卡剩余领取次数
	UserInfo_register_ip            = "register_ip"            //注册IP,玩家注册IP
	UserInfo_accountregister_time   = "accountregister_time"   //账号注册时间,账号注册时间【账号ID（聚合SDK关联ID）注册时间】
	UserInfo_userregister_time      = "userregister_time"      //角色注册时间,角色注册时间
	UserInfo_userlast_active_time   = "userlast_active_time"   //角色最后活跃时间,角色最后活跃时间
	UserInfo_bag_info               = "bag_info"               //玩家物品信息,玩家物品信息
	UserInfo_total_charge           = "total_charge"           //玩家累计充值金额,玩家累计充值金额，仅记录真实充值
	UserInfo_union_id               = "union_id"               //公会ID,公会ID
	UserInfo_currency_info          = "currency_info"          //公会ID,公会ID
)

/*
	日志说明：用户进入游戏，选择区服后，创建角色时触发
	操作类型：Create_Role
*/
const (
	Create_Role_event_id   = "10005"        //事件id,10005
	Create_Role_sex        = "character_id" //性别,性别 1男 2女
	Create_Role_role_name  = "role_name"    //角色名,角色名称
	Create_Role_role_extra = "role_extra"   //其他信息,记录角色职业等其他：
)

/*
	日志说明：用户进入游戏，选择区服后，登录已创建的角色时触发
	操作类型：Role_Login
*/
const (
	Role_Login_event_id               = "10006"                  //事件id,10006
	Role_Login_free_diamond_balance   = "free_diamond_balance"   //免费获取的剩余量,一级代币，免费获取的剩余量
	Role_Login_donate_diamond_balance = "donate_diamond_balance" //充值赠送的剩余量,一级代币，充值赠送的剩余量，如买1000赠送5%，首充双倍等。
	Role_Login_charge_diamond_balance = "charge_diamond_balance" //充值获得的剩余量,一级代币，充值获得的剩余量
	Role_Login_gold_balance           = "gold_balance"           //二级代币余量,二级代币余量
	Role_Login_login_situation        = "login_situation"        //登录场景,登录场景
	Role_Login_role_level             = "role_level"             //角色等级,角色等级
	Role_Login_vip_level              = "vip_level"              //vip,角色vip等级
)

/*
	日志说明：用户退出游戏角色时触发，注：角色退出时，同步记录用户信息
	操作类型：Role_Logout
*/
const (
	Role_Logout_event_id               = "10007"                  //事件id,10007
	Role_Logout_role_level             = "role_level"             //角色等级,角色等级
	Role_Logout_vip_level              = "vip_level"              //vip,角色vip等级
	Role_Logout_free_diamond_balance   = "free_diamond_balance"   //免费获取的剩余量,一级代币，免费获取的剩余量
	Role_Logout_donate_diamond_balance = "donate_diamond_balance" //充值赠送的剩余量,一级代币，充值赠送的剩余量，如买1000赠送5%，首充双倍等。
	Role_Logout_charge_diamond_balance = "charge_diamond_balance" //充值获得的剩余量,一级代币，充值获得的剩余量
	Role_Logout_gold_balance           = "gold_balance"           //二级代币余量,二级代币余量
	Role_Logout_final_scene            = "final_scene"            //登出场景,登出场景
	Role_Logout_final_action           = "final_action"           //登出前操作,登出前操作
	Role_Logout_online_time            = "online_time"            //在线时长(秒),在线时长(秒)
)

/*
	日志说明：用户获得代币时触发，包含金币，钻石，银币等各类资源
	操作类型：Get_Money
*/
const (
	Get_Money_event_id       = "10008"          //事件id,10008
	Get_Money_role_level     = "role_level"     //角色等级,角色等级
	Get_Money_vip_level      = "vip_level"      //vip,vip等级
	Get_Money_before_balance = "before_balance" //获取前余额,获取前余额
	Get_Money_after_balance  = "after_balance"  //获取后余额,获取后余额
	Get_Money_money_type     = "money_type"     //代币类型,代币类型（1：金币，2：钻石，更多货币请CP自行定义，数字依次累加规则）
	Get_Money_reason         = "reason_id"      //原因,代币来源（充值，活动赠送，更多原因请CP自行定义）
	Get_Money_reason_info    = "reason_info"    //任务或奖励等详情JSON根式，如果为空，请填{}
)

/*
	日志说明：用户消耗代币时触发，包含金币，钻石，银币等各类资源
	操作类型：Remove_Money
*/
const (
	Remove_Money_event_id       = "10009"          //事件id,10009
	Remove_Money_role_level     = "role_level"     //角色等级,角色等级
	Remove_Money_vip_level      = "vip_level"      //vip,vip等级
	Remove_Money_before_balance = "before_balance" //消耗前余额,消耗前余额
	Remove_Money_after_balance  = "after_balance"  //消耗后余额,消耗后余额
	Remove_Money_money_type     = "money_type"     //代币类型,代币类型（1：金币，2：钻石，更多货币请CP自行定义，数字依次累加规则）
	Remove_Money_reason         = "reason_id"      //原因,代币来源（物品购买，装备升级，更多原因请CP自行定义）
	Remove_Money_reason_info    = "reason_info"    //任务或奖励等详情JSON根式，如果为空，请填{}
)

/*
	日志说明：统计当前在线角色数
	操作类型：Online_Scene
*/
const (
	Online_Scene_event_id = "10010"    //事件id,10010
	Online_Scene_role_num = "role_num" //在线角色数量,当前时间有多少人在线
)

/*
	日志说明：记录新建角色新手引导完成情况
	操作类型：NewG
*/
const (
	NewG_event_id  = "10011"     //事件id,10011
	NewG_guide_id  = "guide_id"  //引导阶段,新手第一阶段引导，第二阶段引导，如果有更多引导类型，请CP自行定义
	NewG_guide_num = "guide_num" //引导id,某一阶段第几步，可以写步骤编号，或者是直接写中文描述
)

/*
	日志说明：玩家本次操作的行为id
	操作类型：Action_Line
*/
const (
	Action_Line_event_id  = "10012"     //事件id,10012
	Action_Line_action_id = "action_id" //行为id,自定义行为轨迹点
)

/*
	日志说明：用户获得物品时触发，包含英雄，碎片，消耗道具等
	操作类型：Get_Item
*/
const (
	Get_Item_event_id     = "10013"        //事件id,10013
	Get_Item_role_level   = "role_level"   //角色等级,角色等级
	Get_Item_vip_level    = "vip_level"    //vip,角色vip等级
	Get_Item_product_type = "product_type" //物品类型id, 物品类型id
	Get_Item_product_id   = "product_id"   //物品id,物品id
	Get_Item_product_num  = "product_num"  //数量,数量
	Get_Item_reason_id    = "reason_id"    //原因,备注原因等
	Get_Item_reason_info  = "reason_info"  //原因参数,任务或奖励等详情
)

/*
	日志说明：用户消耗物品时触发，包含英雄，碎片，消耗道具等
	操作类型：Remove_Item
*/
const (
	Remove_Item_event_id     = "10014"        //事件id,10014
	Remove_Item_role_level   = "role_level"   //角色等级,角色等级
	Remove_Item_vip_level    = "vip_level"    //vip,角色vip等级
	Remove_Item_product_type = "product_type" //物品类型id, 物品类型id
	Remove_Item_product_id   = "product_id"   //物品id,物品id
	Remove_Item_product_num  = "product_num"  //数量,数量
	Remove_Item_reason_id    = "reason_id"    //原因,备注
	Remove_Item_reason_info  = "reason_info"  //原因参数,任务或奖励等详情
)

/*
	日志说明：用户在商城购买道具时触发
	操作类型：Shop_Buy
*/
const (
	Shop_Buy_event_id    = "10015"       //事件id,10015
	Shop_Buy_role_level  = "role_level"  //角色等级,角色等级
	Shop_Buy_vip_level   = "vip_level"   //vip,vip等级
	Shop_Buy_product_id  = "product_id"  //物品id,物品ID
	Shop_Buy_product_num = "product_num" //数量,物品数量
	Shop_Buy_money_type  = "money_type"  //代币类型,1：金币，2：钻石等，更多货币请CP自行定义，数字依次累加规则
	Shop_Buy_buy_cost    = "buy_cost"    //花费数量,花费数量
	Shop_Buy_shop_type   = "shop_type"   //商店类型,商城A，黑店，如果有更多商城类型，请CP自行定义
)

/*
	日志说明：参与抽奖时记录
	操作类型：ChouJiang
*/
const (
	ChouJiang_event_id     = "10016"        //事件id,10016
	ChouJiang_role_level   = "role_level"   //角色等级,角色等级
	ChouJiang_vip_level    = "vip_level"    //vip,vip等级
	ChouJiang_action_type  = "action_type"  //抽奖类型,1金币单抽2金币十连抽3钻石单抽4钻石十连抽，如果有更多类型，请CP自行定义
	ChouJiang_rewards_info = "rewards_info" //奖励,物品ID:数量，获得多个物品以英文逗号分隔，如：物品ID:数量,物品ID:数量
	ChouJiang_money_type   = "money_type"   //代币类型,1：金币，2：钻石等，更多货币请CP自行定义，数字依次累加规则
	ChouJiang_buy_cost     = "buy_cost"     //花费数量,花费数量
)

/*
	日志说明：角色经验或级别变化时记录；
	操作类型：Exp_Change
*/
const (
	Exp_Change_event_id     = "10017"        //事件id,10017
	Exp_Change_point        = "point"        //类别,1、角色（玩家）；2、vip；3、主线道具（例如卡牌游戏的卡牌、英雄）；…… 其他对象类型依次往后排列
	Exp_Change_point_before = "point_before" //变化前数值,级别、物品或经验变化前数值
	Exp_Change_point_after  = "point_after"  //变化后数值,级别、物品或经验变化后数值
)

/*
	日志说明：角色领取任务时触发
	操作类型：Receive_Quest
*/
const (
	Receive_Quest_event_id  = "10018"     //事件id,10018
	Receive_Quest_task_type = "task_type" //任务类型,1主线任务2支线任务3活动任务4公会任务5情侣任务6……
	Receive_Quest_task_id   = "task_id"   //任务id,任务唯一标识
)

/*
	日志说明：角色完成任务时触发
	操作类型：Finish_Quest
*/
const (
	Finish_Quest_event_id     = "10019"        //事件id,10019
	Finish_Quest_task_type    = "task_type"    //任务类型,1主线任务2支线任务3活动任务4公会任务5情侣任务6……
	Finish_Quest_task_id      = "task_id"      //任务id,任务唯一标识
	Finish_Quest_rewards_info = "rewards_info" //奖励,任务奖励详情
)

/*
	日志说明：角色进入Pve战斗时触发
	操作类型：Instance_PVE
*/
const (
	Instance_PVE_event_id = "10020"    //事件id,10020
	Instance_PVE_pve_type = "pve_type" //副本类型,1普通副本2精英副本3组队副本4世界boss 5……
	Instance_PVE_pve_id   = "pve_id"   //副本id,pve_id
	Instance_PVE_pve_name = "pve_name" //副本名称,副本名称
	Instance_PVE_team_id  = "team_id"  //组队ID,组队ID
	Instance_PVE_lineup   = "lineup"   //阵容,英雄id;英雄id2;……
)

/*
	日志说明：角色完成Pve战斗时触发
	操作类型：Finish_PVE
*/
const (
	Finish_PVE_event_id      = "10021"         //事件id,10021
	Finish_PVE_pve_type      = "pve_type"      //副本类型,1普通副本2精英副本3组队副本4世界boss 5……
	Finish_PVE_pve_id        = "pve_id"        //副本id,pve_id
	Finish_PVE_pve_name      = "pve_name"      //副本名称,副本名称
	Finish_PVE_team_id       = "team_id"       //组队ID,组队ID
	Finish_PVE_iswin         = "iswin"         //战斗结果,是否胜利：战斗结果，0胜利，1失败
	Finish_PVE_lineup        = "lineup"        //阵容,英雄id;英雄id2;……
	Finish_PVE_complete_type = "complete_type" //扫荡类型,扫荡类型：0，手动通关；1，扫荡1次；5，扫荡五次；以游戏实际情况分类
)

/*
	日志说明：角色进入Pvp战斗时触发
	操作类型：Instance_PVP
*/
const (
	Instance_PVP_event_id       = "10022"          //事件id,10022
	Instance_PVP_role_level     = "role_level"     //角色等级,角色等级
	Instance_PVP_lineup         = "lineup"         //阵容,英雄id;英雄id2;……
	Instance_PVP_fight_id       = "fight_id"       //战斗ID,战斗ID
	Instance_PVP_fight_model_id = "fight_model_id" //战斗模式ID,战斗模式ID
	Instance_PVP_fight_type_id  = "fight_type_id"  //战斗类型ID,战斗类型ID
)

/*
	日志说明：角色完成Pvp战斗时触发
	操作类型：Finish_PVP
*/
const (
	Finish_PVP_event_id        = "10023"           //事件id,10023
	Finish_PVP_role_level      = "role_level"      //角色等级,角色等级
	Finish_PVP_lineup          = "lineup"          //阵容,阵容
	Finish_PVP_fight_id        = "fight_id"        //战斗ID,战斗ID
	Finish_PVP_fight_model_id  = "fight_model_id"  //战斗模式ID,战斗模式ID
	Finish_PVP_fight_type_id   = "fight_type_id"   //战斗类型ID,战斗类型ID
	Finish_PVP_map_id          = "map_id"          //场景ID,场景ID
	Finish_PVP_team_player_num = "team_player_num" //组队人数,组队人数
	Finish_PVP_iswin           = "iswin"           //战斗结果,是否胜利0失败1胜利
	Finish_PVP_fight_time      = "fight_time"      //战斗时长（秒）,战斗时长（秒）
	Finish_PVP_rank            = "rank"            //排名,排名
)

/*
	日志说明：到达登陆界面时。登录界面一般指选择登录方式的界面，如选择英雄账号、手机号、其他账号等
	操作类型：User_Login
*/
const (
	User_Login_event_id     = "10004"        //事件id,10004
	User_Login_user_balance = "user_balance" //账号级别代币余额,账号下各类代币持有数量json，如果为空，请填{}
	User_Login_ads_json     = "ads_json"     //联运渠道广告标识
)

/*
	日志说明：用户角色有充值行为时触发
	操作类型：ChargeInfo
*/
const (
	ChargeInfo_event_id       = "10003"          // 事件id,10003
	ChargeInfo_role_name      = "role_name"      // 角色名称,角色名称
	ChargeInfo_role_level     = "role_level"     // 角色等级,角色等级
	ChargeInfo_vip_level      = "vip_level"      // VIP,VIP等级
	ChargeInfo_order_id       = "order_id"       // 聚合SDK订单号,聚合SDK订单号
	ChargeInfo_game_order_id  = "game_order_id"  // 游戏订单号,游戏订单号
	ChargeInfo_product_id     = "product_id"     // 商品id,商品ID
	ChargeInfo_product_type   = "product_type"   // 档位类型,充值档位ID
	ChargeInfo_total_charge   = "total_charge"   // 玩家累计充值金额,充值总和（加上本次充值的）
	ChargeInfo_amount         = "amount"         // 金额,金额
	ChargeInfo_unit           = "unit"           // 货币单位,货币单位 CNY:人民币 USD:美元AUD:欧元 JPY:日元 HKD:港元 GBP：英镑
	ChargeInfo_pay_time       = "pay_time"       // 充值时间,充值时间
	ChargeInfo_get_time       = "get_time"       // 到账时间,到账时间
	ChargeInfo_extra_get1     = "extra_get1"     // 充值获得代币数量,充值获得代币数量（不含赠送代币）
	ChargeInfo_extra_get2     = "extra_get2"     // 充值获得赠送代币数量,充值获得赠送代币数量
	ChargeInfo_money_type     = "money_type"     // 获得代币类型,获得代币类型
	ChargeInfo_payment_method = "payment_method" // 付款方式 1 sdk（常规付款操作）2 虚拟支付（测试用）3英雄线下操作
)

// new gen log
