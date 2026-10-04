package randpool

import (
	"strconv"
	"strings"

	"github.com/nghichtu91/platform/share/planx/rand_pool"
)

type Distribution interface {
	GetRatio() (float64, error)
}

type DensityInfo struct {
	ID     string
	Step   uint32
	Ratios []float64
}

func (d *DensityInfo) parseRatios(rs string) error {
	if rs == "" {
		return ErrInvalidRatioSequence
	}
	rss := strings.Split(rs, ",")
	d.Ratios = make([]float64, len(rss))
	last := 0.0
	for i := 0; i < len(rss); i++ {
		f, err := strconv.ParseFloat(rss[i], 64)
		if err != nil {
			return err
		}
		// 确保后面的数字不能比前面小
		if last > f {
			return ErrInvalidRatioSequence
		}
		last = f
		d.Ratios[i] = f
	}
	return nil
}

/*
	策划需求简述
	-. 生成0～1之间随机数f
	-. 每Step/100为一个区间，判断随机数f落在哪个区间，取向后的序号 e.g. step=1, r=0.5687, 则取第57个数，idx=56
	-. 返回ratios序列中对应序列的值
*/
func (d *DensityInfo) GetRatio() (float64, error) {
	if d.Step == 0 {
		return 0, ErrInvalidStep
	}
	rd := rand_pool.Float64()
	idx := int(rd * 100 / float64(d.Step))
	if idx >= len(d.Ratios) {
		return 0, ErrInvalidRatios
	}
	return d.Ratios[idx+1], nil
}
