package nt

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSign(t *testing.T) {
	req := &PushReq{
		AcID:       "玩家 ACID",
		TemplateID: 1,
		AppID:      1009,
		Macros: map[string]interface{}{
			"GiftID": 1,
		},
		Sign: "778c7fa95c0b845900eb0ea319a02f0b",
	}

	raw := `
{
    "appId": 1009,
    "acid": "玩家 ACID",
    "templateId":1,
    "sign":"778c7fa95c0b845900eb0ea319a02f0b",
    "macros":{
        "GiftID":1
    }
}
`
	req2 := &PushReq{}
	err := json.Unmarshal([]byte(raw), req2)
	assert.Nil(t, err)

	assert.Equal(t, req, req2)

	assert.Equal(t, "778c7fa95c0b845900eb0ea319a02f0b", Sign("eb6cf221fc3a527672c4b44f8a6fc774", req))
	assert.Equal(t, "778c7fa95c0b845900eb0ea319a02f0b", Sign("eb6cf221fc3a527672c4b44f8a6fc774", req2))

}
