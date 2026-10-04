package chatSystem

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	gmConfig "github.com/nghichtu91/platform/share/x/chat/web/config"
	"github.com/nghichtu91/platform/share/x/chat/web/db"
	"github.com/nghichtu91/platform/share/x/chat/web/model/super_user"
)

type SearchCharInfoRequest struct {
	Times          []string `form:"times"`
	GameSerId      string   `form:"gameserver_id"`
	Sender         string   `form:"sender"`
	Receiver       string   `form:"receiver"`
	SenderDevice   string   `form:"sender_device"`
	DataStatus     string   `form:"data_status"`
	Content        string   `form:"content"`
	InterceptCause string   `form:"intercept_cause"`
}

type SearchCharInfoResponse struct {
	ResData []map[string]interface{} `json:"data"`
}

func (req *SearchCharInfoRequest) Handle(c *gin.Context, g super_user.UserInfo) (interface{}, error, string) {
	resp := &SearchCharInfoResponse{}
	var (
		r map[string]interface{}
	)
	es := db.InitEsClient()
	res, err := es.Info()
	// 拼接es查询的body
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []map[string]interface{}{},
			},
		},
		"size": 1000,
	}

	var filterData []map[string]interface{}
	t := reflect.TypeOf(req).Elem()
	v := reflect.ValueOf(req).Elem()
	// 根据前端传递的参数动态生成filter
	for k := 0; k < t.NumField(); k++ {
		if v.Field(k).Interface() != "" {
			if t.Field(k).Tag.Get("form") == "times" {
				if len(v.Field(k).Interface().([]string)) == 2 {
					generalData := map[string]interface{}{
						"range": map[string]interface{}{
							"@timestamp": map[string]interface{}{
								"format": "strict_date_optional_time",
								"gte":    v.Field(k).Interface().([]string)[0],
								"lte":    v.Field(k).Interface().([]string)[1],
							},
						},
					}
					filterData = append(filterData, generalData)
				}
				continue
			}

			generalData := map[string]interface{}{
				"wildcard": map[string]interface{}{
					fmt.Sprintf("%s", t.Field(k).Tag.Get("form")): fmt.Sprintf("*%s*", v.Field(k).Interface().(string)),
				},
			}
			filterData = append(filterData, generalData)
		}
	}
	queryData := query["query"].(map[string]interface{})["bool"].(map[string]interface{})
	queryData["filter"] = filterData

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		tilogs.L().Errorf("InitEsClient error %v")
	}
	tilogs.L().Infof("%v", buf)
	// 查询es
	res, err = db.InitEsClient().Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex(gmConfig.Cfg.GidConfig.ESIndex),
		es.Search.WithBody(&buf),
		es.Search.WithTrackTotalHits(true),
		es.Search.WithPretty(),
	)
	if err != nil {
		return resp.ResData, err, fmt.Sprintf("%v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return resp.ResData, err, fmt.Sprintf("%v", err)
		} else {
			return resp.ResData, err, fmt.Sprintf("%v", e)
		}
	}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return resp.ResData, err, fmt.Sprintf("%v", err)
	}
	// 解析response
	for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
		resp.ResData = append(resp.ResData, hit.(map[string]interface{})["_source"].(map[string]interface{}))
	}
	return resp, nil, ""
}
