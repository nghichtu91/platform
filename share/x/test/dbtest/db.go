package dbtest

import (
	"testing"

	"github.com/gomodule/redigo/redis"
	"github.com/nghichtu91/platform/share/planx/redispool"
)

func DBExists(t *testing.T, pool redispool.IPool, tableName, keyName string, isExist bool) {
	_s1Exist, err := pool.GetDBConn().Do("", "HEXISTS", tableName, keyName)
	if err != nil {
		t.Fatalf("%v %v exists err %v", tableName, keyName, err)
	}
	s1Exist, err := redis.Int(_s1Exist, err)
	if err != nil {
		t.Fatalf("%v %v exists ret to int err %v", tableName, keyName, err)
	}
	if isExist {
		if s1Exist == 0 {
			t.Fatalf("%v %v not exists", tableName, keyName)
		}
	} else {
		if s1Exist == 1 {
			t.Fatalf("%v %v exists", tableName, keyName)
		}
	}

}
