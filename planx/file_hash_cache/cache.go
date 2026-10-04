package file_hash_cache

import (
	"encoding/json"
	"os"

	"github.com/nghichtu91/platform/share/planx/util"
)

// Cache 按执行轮次实现的普通缓存，不支持并发
// 适用于每次使用相对固定的一批文件执行某些操作的逻辑
// 当次操作的所有记录，Cache Close后存入文件
type Cache struct {
	// 缓存存储和读取路径
	path string

	// 文件hash历史缓存
	pre map[string]string

	// 当前文件hash
	cur map[string]string
}

func NewCache(path string) *Cache {
	return &Cache{
		path: path,
		pre:  make(map[string]string, 1024),
		cur:  make(map[string]string, 1024),
	}
}

func (c *Cache) Start() error {
	return c.Load()
}

func (c *Cache) Close() error {
	return c.Save()
}

// Save 将当前文件哈希缓存保存到本地文件
// 不会改变内存里的历史数据和当前数据
func (c *Cache) Save() error {
	raw, err := json.MarshalIndent(c.cur, "", "  ")
	if err != nil {
		return err
	}

	err = os.WriteFile(c.path, raw, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

// Load 从指定文件读取上次保存的哈希缓存
func (c *Cache) Load() error {
	if c.path == "" {
		return ErrCacheFileNameMissing
	}

	if !util.FileExists(c.path) {
		return nil
	}

	raw, err := os.ReadFile(c.path)
	if err != nil {
		return err
	}

	err = json.Unmarshal(raw, &c.pre)
	if err != nil {
		return err
	}

	return nil
}

// UpdateFiles 更新列表里所有文件信息，并返回是否存在文件哈希变化
// 如果文件名为空，会直接跳过此文件，不会报错
// 如果出现异常，不会遍历所有文件
func (c *Cache) UpdateFiles(filenames ...string) (changed bool, err error) {
	var (
		fc bool
	)

	for _, fn := range filenames {
		if fn == "" {
			continue
		}
		fc, err = c.updateFileInfo(fn)
		if err != nil {
			return
		}
		if fc {
			changed = true
		}
	}

	return
}

func (c *Cache) updateFileInfo(filename string) (changed bool, err error) {
	var (
		h string
	)
	h, err = GenHash(filename)
	if err != nil {
		return
	}

	c.cur[filename] = h

	ph, ok := c.pre[filename]
	changed = !ok || ph != h

	return
}

func (c *Cache) GetChangedInfo() *ChangeInfo {
	ci := &ChangeInfo{
		Added:   make([]string, 0, 64),
		Changed: make([]string, 0, 256),
		Deleted: make([]string, 0, 64),
	}

	// 增，改
	for f, h := range c.cur {
		ph, ok := c.pre[f]
		if !ok {
			ci.Added = append(ci.Added, f)
			continue
		}
		if ph != h {
			ci.Changed = append(ci.Changed, f)
			continue
		}
	}

	// 删
	for _, f := range c.pre {
		_, ok := c.cur[f]
		if !ok {
			ci.Deleted = append(ci.Deleted, f)
			continue
		}
	}

	return ci
}
