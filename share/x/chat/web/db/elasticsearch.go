package db

import (
	elasticsearch7 "github.com/elastic/go-elasticsearch/v7"
	_ "github.com/go-sql-driver/mysql"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/chat/web/config"
)

func InitEsClient() *elasticsearch7.Client {
	dbcfg := config.Cfg.GidConfig
	cfg := elasticsearch7.Config{
		Addresses: []string{
			dbcfg.ESUrl,
		},
	}
	es, err := elasticsearch7.NewClient(cfg)
	if err != nil {
		tilogs.L().Errorf("InitEsClient error %v")
	}
	return es
}
