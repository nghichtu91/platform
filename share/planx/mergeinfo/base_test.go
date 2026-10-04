package mergeinfo

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFromJson(t *testing.T) {
	t.Run("parse g2s success", func(t *testing.T) {
		tests := []string{
			`[{"gid":1}]`,
			`[{"gid":1, "sss": "10001"}]`,
			`[{"gid":16, "sss":"160001,160002,160099-160999"},{"gid":17, "sss":"170001,170002,170099-170999"},{"gid":99, "sss":"99999"}]`,
		}

		for i, test := range tests {
			failMsg := "test " + strconv.Itoa(i) + " failed"
			m, err := NewSid2GidInfoFromJson([]byte(test))
			assert.Nil(t, err, failMsg)
			assert.NotNil(t, m, failMsg)
		}
	})

	t.Run("parse g2s fail", func(t *testing.T) {
		tests := []string{
			``,
			`[]`,
			`[{"gid":1, "sss": 10001}]`,
			`[{"gid":1, "sss": "10001, 10001-10001"}]`,
			`[{"gid":16, "sss":"160001,160002,160099-160999"},{"gid":16, "sss":"170001,170002,170099-170999"},{"gid":99, "sss":"99999"}]`,
			`[{"gid":98, "sss":"99999"}, {"gid":99, "sss":"99999"}]`,
			`[{"gid":98, "sss":"88888-90000"}, {"gid":99, "sss":"90000-99999"}]`,
		}

		for i, test := range tests {
			failMsg := "test " + strconv.Itoa(i) + " failed"
			m, err := NewSid2GidInfoFromJson([]byte(test))
			assert.NotNil(t, err, failMsg)
			assert.Nil(t, m, failMsg)
		}
	})

	t.Run("parse shard info success", func(t *testing.T) {
		tests := []string{
			`{"sid": 160001, "sss":"160001"}`,
			`{"sid": 160001, "sss":"160001", "status":0}`,
			`{"sid": 160001, "sss":"160001,160099-160999", "status":1}`,
			`{"sid": 160001, "sss":"160001,160003,160099-160999", "status":2}`,
			`{"sid": 64, "sss":"64", "status":1}`,
		}

		for i, test := range tests {
			failMsg := "test " + strconv.Itoa(i) + " failed"
			m, err := NewShardInfoFromJson([]byte(test))
			assert.Nil(t, err, failMsg)
			assert.NotNil(t, m, failMsg)
		}
	})

	t.Run("parse shard info fail", func(t *testing.T) {
		tests := []string{
			``,
			`{}`,
			`{"sid": 160001}`,
			`{"sid": 160001, "sss":"160002"}`,
			`{"sid": 160002, "sss":"160002,160002"}`,
			`{"sid": 160002, "sss":"160003-160002"}`,
			`{"sss":"160001,160099-160999", "status":1}`,
		}

		for i, test := range tests {
			failMsg := "test " + strconv.Itoa(i) + " failed"
			m, err := NewShardInfoFromJson([]byte(test))
			assert.NotNil(t, err, failMsg)
			assert.Nil(t, m, failMsg)
		}
	})
}

func TestShards_I2S(t *testing.T) {
	t.Run("parse success", func(t *testing.T) {
		tests := []Shards{
			{ShardsStr: "", ShardIDs: nil},
			{ShardsStr: "160001", ShardIDs: []int{160001}},
			{ShardsStr: "160001-160003", ShardIDs: []int{160001, 160002, 160003}},
			{ShardsStr: "160001, 160003-160005", ShardIDs: []int{160001, 160003, 160004, 160005}},
			{ShardsStr: "160001, 160003-160003", ShardIDs: []int{160001, 160003}},
			{ShardsStr: "\"160001, 160003-160005\"", ShardIDs: []int{160001, 160003, 160004, 160005}},
		}

		for i, test := range tests {
			failMsg := "test " + strconv.Itoa(i) + " failed"
			shards, err := test.S2I()
			assert.Equal(t, nil, err, failMsg)
			assert.Equal(t, test.ShardIDs, shards, failMsg)
		}

		for i, test := range tests {
			failMsg := "test " + strconv.Itoa(i) + " failed"
			assert.Equal(t, test.ShardsStr, test.I2S(), failMsg)
		}
	})

	t.Run("parse fail", func(t *testing.T) {
		tests := []Shards{
			{ShardsStr: " 16OO7 ", ShardIDs: nil},
			{ShardsStr: " ae ", ShardIDs: nil},
			{ShardsStr: "16003,16001", ShardIDs: nil},
			{ShardsStr: "16001，16003", ShardIDs: nil},
			{ShardsStr: "160005-16007", ShardIDs: nil},
			{ShardsStr: "16000,16000-16000", ShardIDs: nil},
		}

		for i, test := range tests {
			failMsg := "test " + strconv.Itoa(i) + " failed"
			shards, err := test.S2I()
			assert.Equal(t, ErrInvalidRangeString, err, failMsg)
			assert.Equal(t, test.ShardIDs, shards, failMsg)
		}

		for i, test := range tests {
			failMsg := "test " + strconv.Itoa(i) + " failed"
			assert.Equal(t, test.ShardsStr, test.I2S(), failMsg)
		}
	})
}
