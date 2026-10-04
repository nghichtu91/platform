package core

import "time"

func Run() {
	f := foo{a: 1}
	for {
		MyFunc()
		f.add(1)
		f.hello()

		time.Sleep(5 * time.Second)
	}
}
