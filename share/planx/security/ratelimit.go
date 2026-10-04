package security

import (
	"strconv"
	"strings"
	"time"

	"github.com/nghichtu91/platform/share/planx/limit/ratelimit"
	"github.com/nghichtu91/platform/share/planx/metrics"
	"github.com/nghichtu91/platform/share/planx/tilogs"

	"net"

	gm "github.com/rcrowley/go-metrics"
)

type internalIP struct {
	From net.IP
	To   net.IP
}

var (
	limiter          *ratelimit.TokenStringLimiter
	isValid          bool
	cfgUrl           map[string]*ratelimit.RateSet
	ratelimitCounter gm.Counter
)

func LoadConf(valid bool, urlLimitInfo string) {
	isValid = valid
	if !isValid {
		return
	}

	urlInfos := strings.Split(urlLimitInfo, ";")
	cfgUrl = make(map[string]*ratelimit.RateSet, len(urlInfos))
	for _, url := range urlInfos {
		infos := strings.Split(url, ",")
		if len(infos) < 3 {
			tilogs.L().Errorf("gamex ratelimit url cfg not enough: %s", url)
			continue
		}
		rates := ratelimit.NewRateSet()
		t, err := strconv.ParseInt(infos[1], 10, 64)
		if err != nil {
			tilogs.L().Errorf("gamex ratelimit url cfg time err: %s", url)
			continue
		}
		n, err := strconv.ParseInt(infos[2], 10, 64)
		if err != nil {
			tilogs.L().Errorf("gamex ratelimit url cfg count err: %s", url)
			continue
		}
		rates.Add(time.Second*time.Duration(t), n, n)
		cfgUrl[infos[0]] = rates
	}
	ratelimitCounter = metrics.NewCounter("gamex.ratelimit")
	tilogs.L().Debugf("ratelimite load conf %v", cfgUrl)
}

func Init() {
	if !isValid {
		return
	}

	rates := ratelimit.NewRateSet()
	rates.Add(time.Second, 1, 10)

	stringExtractRates := func(s string) (*ratelimit.RateSet, error) {
		rates, ok := cfgUrl[getUrl(s)]
		if ok {
			return rates, nil
		}
		// 找不到则不限制
		return nil, nil
	}

	l, err := ratelimit.NewForString(rates, ratelimit.StringExtractRates(ratelimit.StringRateExtractorFunc(stringExtractRates)))
	if err != nil {
		tilogs.L().Errorf("gamex init ratelimit err: %s", err.Error())
		return
	}
	limiter = l
}

func Consume(source string) error {
	if !isValid {
		return nil
	}
	if err := limiter.Consume(source); err != nil {
		ratelimitCounter.Inc(1)
		return err
	}
	return nil
}

func GenSource(acid string, url string) string {
	return acid + "@" + url
}

func getUrl(source string) string {
	ss := strings.Split(source, "@")
	return ss[1]
}
