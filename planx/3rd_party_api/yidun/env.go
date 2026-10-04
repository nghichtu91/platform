package yidun

var (
	cfg *yiDunConfig
)

type yiDunConfig struct {
	businessId string
	secretID   string
	secretKey  string
	ver        string
	url        string
}

// InitYiDun 初始化易盾环境变量
func InitYiDun(businessId, secretID, secretKey, ver, url string) {
	cfg = &yiDunConfig{
		businessId: businessId,
		secretID:   secretID,
		secretKey:  secretKey,
		ver:        ver,
		url:        url,
	}
}
