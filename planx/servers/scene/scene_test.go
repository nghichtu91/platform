package scene

import (
	"fmt"
	"testing"
)

func TestScene(t *testing.T) {
	res := getDynamicSceneUniq("371012:201", 0)
	fmt.Printf("res:%v \n", res)
	res = getDynamicSceneUniq("371012:201", 10086)
	fmt.Printf("res:%v \n", res)
	res = getDynamicSceneUniq("371012:201:10086", 0)
	fmt.Printf("res:%v \n", res)
	res = getDynamicSceneUniq("371012:201:10086", 10086)
	fmt.Printf("res:%v \n", res)
}
