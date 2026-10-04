package metrics

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/atomic"
)

type CommandMetrics struct {
	setOnce sync.Once     // set的标记
	cache   sync.Map      // 命令的缓存
	prefix  atomic.String // 应对可能的race
}

// 不能在命令、前缀中 添加 除大小写字母、数字外的特殊符号。
func (c *CommandMetrics) SetGamexPrefix(gid, shardID uint32, prefix string) {
	c.SetPrefix(fmt.Sprintf("requests.%d.%d.%s", gid, shardID, prefix))
}

// 不能在命令、前缀中 添加 除大小写字母、数字外的特殊符号。
func (c *CommandMetrics) SetCrossxPrefix(gid, serverID uint32, prefix string) {
	c.SetPrefix(fmt.Sprintf("requests.%d.%d.%s", gid, serverID, prefix))
}

//SetPrefix 设置前缀, 只能设置一次
func (c *CommandMetrics) SetPrefix(prefix string) bool {
	var valid bool
	c.setOnce.Do(func() {
		valid = true
		c.prefix.Store(prefix)
	})
	return valid
}

func (c *CommandMetrics) getMetricName(cmd string) string {
	metricName, ok := c.cache.Load(cmd)
	if !ok {
		// 多线程环境下, 存在一定的可能, 多次生成val
		metricName, _ = c.cache.LoadOrStore(cmd, c.prefix.String()+cmd)
	}
	return metricName.(string)
}

func (c *CommandMetrics) time(metricName, key4stat string, start time.Time) {
	TimeStatistics(metricName, key4stat, start.UnixNano())
}

func (c *CommandMetrics) count(metricName string) {
	CountStatistics(metricName)
}

//CountTime start是开始时间
// 不能在命令、前缀中 添加 除大小写字母、数字外的特殊符号。
func (c *CommandMetrics) CountTime(cmd string, start time.Time) {
	metricName := c.getMetricName(cmd)
	c.time(metricName, cmd, start)
	c.count(metricName)
}

func (c *CommandMetrics) Time(cmd string, start time.Time) {
	c.time(c.getMetricName(cmd), cmd, start)
}

func (c *CommandMetrics) Count(cmd string) {
	c.count(c.getMetricName(cmd))
}
