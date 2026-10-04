package shardId

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/chat/web/config"
	"github.com/nghichtu91/platform/share/x/chat/web/model/super_user"
	"github.com/nghichtu91/platform/share/x/common/shard"
)

type GetShardRequest struct {
}

type GetShardResponse struct {
	List []*ShardInfo `json:"list"`
}

type ShardInfo struct {
	ShardId   uint   `json:"shardId"`
	ShardName string `json:"shardName"`
}

func (req *GetShardRequest) Handle(c *gin.Context, g super_user.UserInfo) (interface{}, error, string) {
	resp := &GetShardResponse{}
	var err error
	var errMsg string
	resp.List, errMsg = GetAllShards()
	return resp, err, errMsg
}

func GetAllShards() ([]*ShardInfo, string) {
	sidMaps := shard.LoadShardInfo(config.Cfg.DevopsEtcdRoot, config.Cfg.ServerEtcdRoot, fmt.Sprint(config.Cfg.LocalConfig.Gid))
	sids := make([]*ShardInfo, 0, len(sidMaps))

	for sid, v := range sidMaps {
		if v.SName == "" {
			continue
		}
		sidUint, err := strconv.Atoi(sid)
		if err != nil {
			tilogs.L().Errorf("sid error %v", err)
			continue
		}
		shardName := strings.Split(v.DisplayName, "-")[1]
		sids = append(sids, &ShardInfo{
			ShardId:   uint(sidUint),
			ShardName: shardName,
		})
	}

	sort.Slice(sids, func(i, j int) bool {
		return sids[i].ShardId < sids[j].ShardId
	})

	return sids, ""
}
