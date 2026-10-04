package file_hash_cache

import (
	"encoding/json"
	"os"

	"github.com/nghichtu91/platform/share/planx/util"
)

// DCache 累积缓存
// 每次进行操作，都会修改当前缓存的值
// 缓存中会存储所有访问过的文件历史
// 适用于访问文件整体规模相对固定，但每次访问数量不固定的场景
// 不支持并发
type DCache struct {
	path string

	hash map[string]string
}

func NewDCache(path string) *DCache {
	return &DCache{
		path: path,
		hash: make(map[string]string, 1024),
	}
}

func (dc *DCache) Start() error {
	return dc.Load()
}

func (dc *DCache) Close() error {
	return dc.Save()
}

// Save 将当前文件哈希缓存保存到本地文件
// 不会改变内存里的历史数据和当前数据
func (dc *DCache) Save() error {
	raw, err := json.MarshalIndent(dc.hash, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(dc.path, raw, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

// Load 从指定文件读取上次保存的哈希缓存
func (dc *DCache) Load() error {
	if dc.path == "" {
		return ErrCacheFileNameMissing
	}

	if !util.FileExists(dc.path) {
		return nil
	}

	raw, err := os.ReadFile(dc.path)
	if err != nil {
		return err
	}

	err = json.Unmarshal(raw, &dc.hash)
	if err != nil {
		return err
	}

	return nil
}

// UpdateFiles 更新列表里所有文件信息，并返回是否存在文件哈希变化
func (dc *DCache) UpdateFiles(filenames ...string) (changed bool, err error) {
	var (
		fc bool
	)

	for _, fn := range filenames {
		if fn == "" {
			continue
		}
		fc, err = dc.updateFileInfo(fn)
		if err != nil {
			return
		}
		if fc {
			changed = true
		}
	}

	return
}

func (dc *DCache) updateFileInfo(filename string) (changed bool, err error) {
	var (
		h string
	)
	h, err = GenHash(filename)
	if err != nil {
		return
	}

	ph, ok := dc.hash[filename]
	changed = !ok || ph != h

	dc.hash[filename] = h

	return
}

// GetChangedInfo 获取当前文件哈希和历史文件哈希对比信息
// DCache 不支持此接口，固定返回nil
func (dc *DCache) GetChangedInfo() *ChangeInfo {
	return nil
}
