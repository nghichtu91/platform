package multilingual

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveJsonSrvIdx(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"", ""},
		{"{}", "{}"},
		{"{\"common\": \"1.S1-桃源村\"}", "{\"common\":\"S1-桃源村\"}"},
		{"{\"zh-CN\":\"38.S38-烽火燎原\",\"zh-TW\":\"38.S38-烽火燎原\",\"en-US\":\"38.S38-Flames of War\",\"id-ID\":\"38.S38-Api Peperangan\"}",
			"{\"zh-CN\":\"S38-烽火燎原\",\"zh-TW\":\"S38-烽火燎原\",\"en-US\":\"S38-Flames of War\",\"id-ID\":\"S38-Api Peperangan\"}"},
	}

	for i, test := range tests {
		assert.Equal(t, test.out, RemoveJsonSrvIdx(test.in), i)
	}
}
