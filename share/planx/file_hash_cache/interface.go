package file_hash_cache

type ICache interface {
	Start() error

	Close() error

	// Save 将当前文件哈希缓存保存到本地文件
	// 不会改变内存里的历史数据和当前数据
	Save() error

	// Load 从指定文件读取上次保存的哈希缓存
	Load() error

	// UpdateFiles 更新列表里所有文件信息，并返回是否存在文件哈希变化
	UpdateFiles(filenames ...string) (changed bool, err error)

	// GetChangedInfo 获取当前文件哈希和历史文件哈希对比信息
	GetChangedInfo() *ChangeInfo
}

// ChangeInfo 对比当前和历史数据，列出所有变化的文件
type ChangeInfo struct {
	Added   []string
	Changed []string
	Deleted []string
}
