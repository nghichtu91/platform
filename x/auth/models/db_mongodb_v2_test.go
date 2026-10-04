package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDBByMongoDBV2_Init(t *testing.T) {
	mdbv2 := &DBByMongoDBV2{}
	assert.Nil(t, mdbv2.Init(DBConfig{
		MongoDBName: "AuthDB",
		MongoDBUrl:  "127.0.0.1:27017",
	}))

	v := &MongoAuth{
		UserID:       "c4c61be3-f849-44ca-9a83-cea85ad54d48",
		DisplayName:  "anonymous",
		LastGidSid:   "12:1",
		Device:       "heihei",
		LastAuthTime: 1626333735,
		CreateTime:   1626333735,
		NameAuth:     "jy0001",
		NameAuthPwd:  "c355d4b7ab61ce6766783d2d738e36b1",
	}

	assert.Nil(t, mdbv2.updateAtomicWith(mdbv2.cm.Devices, "user_id", "c4c61be3-f849-44ca-9a83-cea85ad54d48", 2, v))
}
