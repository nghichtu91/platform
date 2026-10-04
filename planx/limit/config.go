package limit

import "net"

type IpRange struct {
	From string `toml:"from"`
	To   string `toml:"to"`
}

type internalIP struct {
	From net.IP
	To   net.IP
}

type LimitConfig struct {
	LimitUrlRegex    string    `toml:"limit_url_regex"`
	InternalIPs      []IpRange `toml:"InternalIPs"`
	RateLimitValid   bool      `toml:"rate_limit_valid"`
	RateLimitAverage int64     `toml:"rate_limit_Average"`
	RateLimitBurst   int64     `toml:"rate_limit_Burst"`
}

var (
	LimitCfg LimitConfig
)
