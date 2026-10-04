package super_user

import (
	"encoding/json"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

var AdminRootPermission map[int][]int

type UserInfo struct {
	UserName string
}

const (
	AccountGroupAdmin = 0 // 管理员权限
	DefaultPwd        = "123456"
	AdminRoot         = "admin@taiyouxi.cn"
	AdminPwd          = "admin123456" // TODO
	AdminRoleName     = "超级管理员"
)

type Permission map[int][]int

func ParsePermission(bytes []byte) map[int][]int {
	var permission Permission
	if bytes != nil {
		err := json.Unmarshal(bytes, &permission)
		if err != nil {
			tilogs.L().Errorf("parse error %v", err)
			return nil
		}
	}
	return permission
}

// function permission
// menu 的ID是整百， 子功能的ID是以menu递增
const (
	FPDashboard = 1 // 主页

	FPAccountMenu       = 100 // 账号menu
	FPAccountUSER       = 101 // 用户管理
	FPAccountGROUP      = 102 // 权限分组
	FPAccountRECORD     = 103 // 操作记录
	FPChatSystem        = 200 //聊天系统
	FPSensitiveWords    = 201 // 敏感词文件
	FPSearchChatInfo    = 202 // 查询聊天信息
	FPEditConfig        = 203 // 修改etcd配置
	FPEditChatStoreTime = 204 //// 修改聊天模块保存时间
)

// operate permission
const (
	OPQueryMenu = iota + 1 // 1: "查看菜单"
	OPQueryPage            // 2: "查看页面"
	OPQuery                // 3: "查询"
	OPAdd                  // 4: "新增"
	OPUpdate               // 5: "修改"
	OPDelete               // 6: "删除"
	OPCancel               // 7: "取消"
	OPRelease              // 8: "发布"
	OPUpload               // 9: "上传"
	OPStatus               // 10: "状态"
)

func init() {
	AdminRootPermission = map[int][]int{
		FPDashboard:         {OPQueryMenu, OPQueryPage},
		FPAccountMenu:       {OPQueryMenu},
		FPAccountUSER:       {OPQueryMenu, OPQueryPage, OPQuery, OPAdd, OPUpdate, OPDelete, OPCancel, OPRelease, OPUpload, OPStatus},
		FPAccountGROUP:      {OPQueryMenu, OPQueryPage, OPQuery, OPAdd, OPUpdate, OPDelete},
		FPAccountRECORD:     {OPQueryMenu, OPQueryPage, OPQuery, OPAdd, OPUpdate, OPDelete},
		FPChatSystem:        {OPQueryMenu},
		FPSensitiveWords:    {OPQueryMenu, OPQueryPage, OPQuery, OPAdd, OPUpdate, OPDelete},
		FPSearchChatInfo:    {OPQueryMenu, OPQueryPage, OPQuery, OPAdd, OPUpdate, OPDelete},
		FPEditConfig:        {OPQueryMenu, OPQueryPage, OPQuery, OPAdd, OPUpdate, OPDelete},
		FPEditChatStoreTime: {OPQueryMenu, OPQueryPage, OPQuery, OPAdd, OPUpdate, OPDelete},
	}
}
