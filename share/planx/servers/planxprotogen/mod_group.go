package planxprotogen

import (
	"strconv"
)

type ModGroup struct {
	ModuleName string // 模块名
	ActID      uint32 // 跨服运营活动ID
	Group      uint32 // 分组id
}

func (mg ModGroup) GetModuleName() string {
	return mg.ModuleName
}

func (mg ModGroup) GetGroup() uint32 {
	return mg.Group
}

func (mg ModGroup) GetActivityID() uint32 {
	return mg.ActID
}

func (mg ModGroup) GetActivityModuleName() string {
	return mg.ModuleName + "_" + strconv.Itoa(int(mg.ActID))
}

func (mg ModGroup) String() string {
	// cross leader -> member分发过程中，会使用mg.String()作为唯一ID
	// 运营活动底层升级后支持一个分组里使用多个ActID，需要区分同一分组的不同运营活动
	if mg.ActID != 0 {
		return mg.GetActivityModuleName() + "." + strconv.Itoa(int(mg.Group))
	} else {
		return mg.ModuleName + "." + strconv.Itoa(int(mg.Group))
	}
}
