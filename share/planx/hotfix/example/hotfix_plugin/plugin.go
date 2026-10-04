package main

import "fmt"

func MyFunc() {
	fmt.Println("new global func")
}

type foo struct {
	a int
}

func MyAdd(u *foo, b int) {
	u.a += 2 * b
	fmt.Printf("new instance func.  foo.a=%d\n", u.a)
}

func GetHotfixFuncList() map[string]string {
	//map的key是本补丁文件中新函数的名称
	//map的value是旧的可执行文件中的名称(应该用nm来查)
	return map[string]string{
		"MyFunc": "github.com/nghichtu91/platform/share/planx/hotfix/example/core.MyFunc",
		"MyAdd":  "github.com/nghichtu91/platform/share/planx/hotfix/example/core.(*foo).add",
	}
}
