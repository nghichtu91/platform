package etcd

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBind(t *testing.T) {
	// string
	k, v := "/str", "test_str"
	err := Put(k, v)
	if err != nil {
		t.Fatal(err)
	}
	var strRes string
	err = Bind(k, &strRes)
	if err != nil || strRes != v {
		t.Fatal(err, strRes)
	}

	// int
	intK, intV := "/number", 1234
	err = Put(intK, fmt.Sprint(intV))
	if err != nil {
		t.Fatal(err)
	}
	var intRes int
	err = Bind(intK, &intRes)
	if err != nil || intV != intRes {
		t.Fatal(err, intRes)
	}

	// struct 1
	type AB struct {
		A int    `etcd3:"a"`
		B string `etcd3:"b"`
	}
	structK, structV := "/struct", AB{11, "aaa"}
	err = Put(structK+"/a", "11")
	if err != nil {
		t.Fatal(err)
	}
	err = Put(structK+"/b", "aaa")
	if err != nil {
		t.Fatal(err)
	}
	var structRes AB
	err = Bind(structK, &structRes)
	if err != nil || structV != structRes {
		t.Fatal(err, structRes)
	}

	// struct 2
	type ABC struct {
		AB  AB  `etcd3:"ab"`
		ABP *AB `etcd3:"abp"`
	}
	structCK := "/structC"
	err = Put(structCK+"/ab/a", "11")
	if err != nil {
		t.Fatal(err)
	}
	err = Put(structCK+"/ab/b", "111")
	if err != nil {
		t.Fatal(err)
	}
	err = Put(structCK+"/abp/a", "22")
	if err != nil {
		t.Fatal(err)
	}
	err = Put(structCK+"/abp/b", "222")
	if err != nil {
		t.Fatal(err)
	}
	var structCRes ABC
	err = Bind(structCK, &structCRes)
	if err != nil {
		t.Fatal(err)
	}
	if structCRes.AB.A != 11 ||
		structCRes.AB.B != "111" ||
		structCRes.ABP.A != 22 ||
		structCRes.ABP.B != "222" {
		t.Fatal(structCRes)
	}

	// struct JSON
	type ABJSON struct {
		AB AB `etcd3:"ab,json"`
	}
	structJson := "/structJson"
	var Ab = AB{44, "444"}
	jsonBytes, err := json.Marshal(Ab)
	if err != nil {
		t.Fatal(err)
	}
	err = Put(structJson+"/ab", string(jsonBytes))
	if err != nil {
		t.Fatal(err)
	}
	var abJsonRes ABJSON
	err = Bind(structJson, &abJsonRes)
	if err != nil {
		t.Fatal(err)
	}
	if abJsonRes.AB.A != 44 || abJsonRes.AB.B != "444" {
		t.Fatal(err)
	}

	// MAP
	abMapRes := make(map[string]AB)
	err = Bind("/structC", abMapRes)
	if err != nil {
		t.Fatal(err)
	}
	if abMapRes["ab"].A != 11 || abMapRes["ab"].B != "111" ||
		abMapRes["abp"].A != 22 ||
		abMapRes["abp"].B != "222" {
		t.Fatal(abMapRes)
	}
}

type Emb struct {
	EA string
	eb int
	EC string `etcd3:"ec"`
}

func TestSyncObj(t *testing.T) {
	emb := struct {
		A string
		b int
		C string `etcd3:"c"`
	}{
		"a", 11, "c",
	}

	embb := Emb{
		"aa", 22, "cc",
	}

	obj := &struct {
		OA  int `etcd3:"oa"`
		OB  string
		EMB struct {
			A string
			b int
			C string `etcd3:"c"`
		}
		Emb
		OtherEmb Emb `etcd3:"emb_3" json:"emb_j"`
	}{
		22, "OB", emb, embb, embb,
	}

	assert.Nil(t, SyncObj("root/server/test/obj", obj))

	// struct
	v, err := Get("root/server/test/obj/EMB/A")
	assert.Nil(t, err)
	assert.Equal(t, "a", v)

	// struct
	v, err = Get("root/server/test/obj/EMB/c")
	assert.Nil(t, err)
	assert.Equal(t, "c", v)

	// 内置Struct
	v, err = Get("root/server/test/obj/Emb/EA")
	assert.Nil(t, err)
	assert.Equal(t, "aa", v)

	// 内置Struct
	v, err = Get("root/server/test/obj/Emb/ec")
	assert.Nil(t, err)
	assert.Equal(t, "cc", v)

	// 变量
	v, err = Get("root/server/test/obj/oa")
	assert.Nil(t, err)
	assert.Equal(t, "22", v)

	// 变量
	v, err = Get("root/server/test/obj/OB")
	assert.Nil(t, err)
	assert.Equal(t, "OB", v)

	// json struct
	v, err = Get("root/server/test/obj/emb_3")
	assert.Nil(t, err)
	assert.Equal(t, `{"EA":"aa","EC":"cc"}`, v)

	// av, err := GetSubRecursive("root/server/test/obj")
	// assert.Nil(t, err)
	// t.Logf("av: %+v", av)

	assert.Nil(t, DeleteRecursive("root/server/test/obj"))
}
