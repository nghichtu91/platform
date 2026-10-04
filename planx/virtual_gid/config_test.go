package virtual_gid

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigJson(t *testing.T) {
	vg := &VirtualGroupCfg{
		Groups: []VirtualGroup{
			{
				VGroupID:     1,
				VGroupName:   "虚拟大区A",
				ShowOffSet:   120000,
				BeginShardID: 120001,
				EndShardID:   125000,
				Channels:     []int{32, 33, 34},
			},
			{
				VGroupID:     2,
				VGroupName:   "虚拟大区B",
				ShowOffSet:   125000,
				BeginShardID: 125001,
				EndShardID:   129000,
				Channels:     []int{2, 3, 5, 7, 18, 63, 127},
			},
			{
				VGroupID:     3,
				VGroupName:   "虚拟大区C",
				ShowOffSet:   129000,
				BeginShardID: 129001,
				EndShardID:   129999,
				Channels:     []int{2, 3, 5, 7, 18, 32, 33, 34, 63, 127},
			},
		}}

	bs, err := json.MarshalIndent(vg, "", "  ")
	assert.Nil(t, err)

	t.Logf("Virtual Group json: %s", string(bs))
}
