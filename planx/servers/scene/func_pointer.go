package scene

import "fmt"

type GetGameDataValue func(levelId int32) int

var getMaxPlayerScene GetGameDataValue
var getNeedNewScene GetGameDataValue

func InitDataValuePointer(pointerType uint8, funcPointer GetGameDataValue) {
	if funcPointer == nil {
		panic(fmt.Sprintf("InitDataValuePointer failed funcPointer Must not nil %v %v", pointerType, pointerType))
	}
	switch pointerType {
	case GetMaxPlayerScene:
		getMaxPlayerScene = funcPointer
		return
	case GetNeedNewScene:
		getNeedNewScene = funcPointer
		return
	}
	panic(fmt.Sprintf("InitDataValuePointer failed unKnown %v", pointerType))
}
