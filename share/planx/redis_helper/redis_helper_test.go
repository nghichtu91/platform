package redis_helper

import (
	"fmt"
	"testing"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"

	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
)

func TestDumpToHashDBCmcBufferCheckDirty(t *testing.T) {
	conn, err := redis.Dial("tcp", "127.0.0.1:6379")
	checkPanic(err)

	type RedisTest struct {
		AAA string `json:"AAA"`
	}

	m := map[string]*RedisTest{
		"111": {
			AAA: "aaa",
		},
		"222": {
			AAA: "bbb",
		},
		"333": {
			AAA: "ccc",
		},
	}

	conn.Send("MULTI")
	err, dirty, changed := DumpToHashDBCmcBufferCheckDirty(conn, "key", m, nil)
	checkPanic(err)
	fmt.Println(dirty)
	fmt.Println(changed)
	_, err = conn.Do("EXEC")
	checkPanic(err)
	fmt.Println("init------set 111 222 333")
	assert.Equal(t, []string{"111", "222", "333"}, changed)

	m["333"].AAA = "ddd"
	m["444"] = &RedisTest{
		AAA: "eee",
	}
	conn.Send("MULTI")
	err, dirty, changed = DumpToHashDBCmcBufferCheckDirty(conn, "key", m, dirty)
	checkPanic(err)
	fmt.Println(dirty)
	fmt.Println(changed)
	_, err = conn.Do("EXEC")
	checkPanic(err)
	fmt.Println("update------set 333 444")
	assert.Equal(t, []string{"333", "444"}, changed)

	m2 := make(map[string]*RedisTest)
	RestoreFromHashDB(conn, "key", &m2, false, false)
	dirty = GenDirtyHash(m2)
	fmt.Println(m2)
	fmt.Println(dirty)
	fmt.Println(m2["444"].AAA)
	fmt.Println("load------")
	assert.Equal(t, 4, len(m2))

	delete(m2, "111")
	conn.Send("MULTI")
	err, dirty, changed = DumpToHashDBCmcBufferCheckDirty(conn, "key", m2, dirty)
	checkPanic(err)
	_, err = conn.Do("EXEC")
	checkPanic(err)
	fmt.Println(m2)
	fmt.Println(dirty)
	fmt.Println(changed)
	fmt.Println("del------del 111")
	assert.Equal(t, []string{"111"}, changed)

	m3 := make(map[string]*RedisTest)
	RestoreFromHashDB(conn, "key", &m3, false, false)
	dirty = GenDirtyHash(m3)
	fmt.Println(m3)
	fmt.Println(dirty)
	_, ok := m3["111"]
	fmt.Println(ok)
	fmt.Println("load------")
	assert.Equal(t, false, ok)

	type AAA struct {
		A string
	}
	m4 := map[AAA]int{}
	m5 := map[*AAA]int{}
	err, _, _ = DumpToHashDBCmcBufferCheckDirty(conn, "aaa", m4, nil)
	assert.NotNil(t, err)
	fmt.Println(err)
	err, _, _ = DumpToHashDBCmcBufferCheckDirty(conn, "bbb", m5, nil)
	assert.NotNil(t, err)
	fmt.Println(err)
}

func checkPanic(err error) {
	if nil != err {
		panic(err)
	}
}

// 测试一个struct中的map是不是每次都是dirty
func TestMapInStruct(t *testing.T) {
	zaplog.InitZapLog("", nil, "")
	defer tilogs.Close()
	type SampleStruct struct {
		X          int            `redis:"x"`
		SampleMaps map[string]int `redis:"sample_map"`
	}
	sample := SampleStruct{
		X:          0,
		SampleMaps: map[string]int{"first": 1, "second": 2, "third": 3},
	}

	conn, err := redis.Dial("tcp", "127.0.0.1:6379")
	checkPanic(err)

	err, dirty, _ := DumpToHashDBCmcBufferCheckDirty(conn, "sample_key", sample, nil)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	for i := 0; i < 100; i++ {
		err, _, dirtyKeys := DumpToHashDBCmcBufferCheckDirty(conn, "sample_key", sample, dirty)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		if len(dirtyKeys) != 0 {
			t.Errorf("dirty %v", dirtyKeys)
			t.FailNow()
		}
	}
}
