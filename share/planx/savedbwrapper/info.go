package savedbwrapper

import (
	"github.com/golang/protobuf/proto"
)

type DBCmd int

const (
	_ DBCmd = iota
	DBCmdAddUpdate
	DBCmdDel
	DBCmdDelTable
	DBCmdExpire
	DBCmdGetAll
)

type IChanger interface {
	SetChange(key string, data proto.Message, dbCmd DBCmd)
}

type ISubChanger interface {
	SetSubChange()
}

type ISub interface {
	IChanger
	GetDBName() string
	Load() error
}

type DBMsgInfo struct {
	DBCmd
	SubKey string
	Data   proto.Message
	Params []string

	acid   string
	hooker IMarshalHook
}

type DBDataInfo struct {
	DBCmd
	SubKey  string
	Data    []byte
	Params  []string
	ChReply chan interface{}
}

type DBDataInfoWithKey struct {
	Key string
	*DBDataInfo
}

type hgetallInfo struct {
	i       int
	chReply chan interface{}
}
