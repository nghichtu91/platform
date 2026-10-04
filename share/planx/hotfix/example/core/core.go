package core

import "fmt"

//这里的都是原函数

func MyFunc() {
	fmt.Println("old global func")
}

type foo struct {
	a int
}

func (u *foo) add(b int) {
	u.a += b
	fmt.Printf("old instance func.  foo.a=%d\n", u.a)
}

func (u *foo) hello() {
	fmt.Println("hi")
}
