package savedbwrapper

import (
	"github.com/golang/protobuf/proto"
)

/*
	此文件用于db数据和存储是不同goroutine的情况，大部分情况应该用不到
*/
type changeInfo struct {
	DBCmd
	DBFieldName    string
	SubDbFieldName string
	Data           proto.Message
	Params         []string

	Acid   string
	Hooker IMarshalHook
}

type IMarshalHook interface {
	BeforeMarshal(acid string)
	AfterMarshal(acid string)
}

func (save *SaveDB) SendChanged(dbCmd DBCmd, dbFieldName, subDbFieldName string, acid string, hooker IMarshalHook, data proto.Message, params ...string) {
	c := &changeInfo{
		DBCmd:          dbCmd,
		DBFieldName:    dbFieldName,
		SubDbFieldName: subDbFieldName,
		Data:           data,
		Params:         params,
		Acid:           acid,
		Hooker:         hooker,
	}
	save.changeChASync <- c // 这里是没有超时的，若底层出问题就卡住，保证数据不会丢
}

func (save *SaveDB) RecvChangedChan() <-chan *changeInfo {
	return save.changeChASync
}
