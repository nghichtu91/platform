package merge_db

import (
	"github.com/nghichtu91/platform/share/planx/redispool"
	"github.com/nghichtu91/platform/share/planx/savedbwrapper"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

var (
	ModuleOpMap          *MergeOpMap     // 模块合服函数映射
	ProfileOpMap         *MergeOpMap     // 存档合服函数映射
	AfterModuleSaveOpMap *MergeOpMap     // 存档写入成功回调的函数映射
	GenDBFnMap           *DBInitMap      // 生成DB函数映射
	GenRenameDB          NewRenameDBFunc // 生成改名DB
	RbOpMap              *RollbackOpMap  // 回滚功能函数映射
)

func init() {
	ModuleOpMap = NewMergeOpMap()
	ProfileOpMap = NewMergeOpMap()
	AfterModuleSaveOpMap = NewMergeOpMap()
	GenDBFnMap = NewDBInitMap()
	RbOpMap = NewRollbackOpMap()
}

// MergeOpMap 关联模块和对应的合并操作函数
type MergeOpMap struct {
	OpFunc map[string]MergeOpFunc // [模块名称]MergeOpFunc
}

// InitMergeDBParams 生成MergeDB的参数列表
type InitMergeDBParams struct {
	ShardID  uint
	MergeUID int
	NeedSave bool
	Pool     redispool.IPool
}

// NewIMergeDBFunc 提供给合服工具统一生成的新合服接口
type NewIMergeDBFunc func(*InitMergeDBParams) IMergeDB

// GenMergeDBFunc 代码生成工具生成的数据库函数接口，如果模块没有对应代码，检查生成工具中，need_merge是否为true
type GenMergeDBFunc func(uint, int, string, string, *savedbwrapper.SaveDB, redispool.IPool, string) IMergeDB

// RollbackOpMap 回滚功能接口
type RollbackOpMap struct {
	OpFunc map[string]RollbackOpFunc // [模块名称]MergeOpFunc RollbackOpFunc
}

// RollbackOpFunc 回滚逻辑实现
type RollbackOpFunc func(*MergeSet) error

// DBInitMap 关联模块和生成函数
type DBInitMap struct {
	GenDBFunc map[string]NewIMergeDBFunc
}

// NewRenameDBFunc 提供给合服工具统一生成的新改名DB生成接口
type NewRenameDBFunc func(*InitMergeDBParams) ICMDLoader

type PreOps []MergeOpFunc

func NewMergeOpMap() *MergeOpMap {
	return &MergeOpMap{
		OpFunc: make(map[string]MergeOpFunc),
	}
}

func NewDBInitMap() *DBInitMap {
	return &DBInitMap{
		GenDBFunc: make(map[string]NewIMergeDBFunc),
	}
}

func NewRollbackOpMap() *RollbackOpMap {
	return &RollbackOpMap{
		OpFunc: make(map[string]RollbackOpFunc),
	}
}

// RegOpFunc 将模块与合服操作关联
func (mm *MergeOpMap) RegOpFunc(name string, fn MergeOpFunc) {
	_, ok := mm.OpFunc[name]
	if ok {
		tilogs.L().Errorf("MergeOpMap OpFunc name %s duplicate", name)
		return
	}
	mm.OpFunc[name] = fn
}

// RegGenFunc 将模块与初始化合服DB关联
func (dm *DBInitMap) RegGenFunc(name string, fn NewIMergeDBFunc) {
	_, ok := dm.GenDBFunc[name]
	if ok {
		tilogs.L().Errorf("DBInitMap RegGenFunc name %s duplicate", name)
		return
	}
	dm.GenDBFunc[name] = fn
}

// RegRollbackFunc 注册合服回滚功能
func (rm *RollbackOpMap) RegRollbackFunc(name string, fn RollbackOpFunc) {
	_, ok := rm.OpFunc[name]
	if ok {
		tilogs.L().Errorf("RollbackOpMap OpFunc name %s duplicate", name)
		return
	}
	rm.OpFunc[name] = fn
}
