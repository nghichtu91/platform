package controllers

import (
	"fmt"
	"testing"
)

func TestMakeData(t *testing.T) {
	str := makeData(map[string]interface{}{
		"key2": 2,
		"key1": 1,
	})
	fmt.Println(str)
	if str != "eyJrZXkyIjoyLCJrZXkxIjoxfQ==" {
		t.FailNow()
	}
}
