package json2proto

import (
	"encoding/json"
	"os"

	"github.com/golang/protobuf/proto"
)

// Json2Proto 将指定json文件反序列化到protobuf结构
func Json2Proto(fn string, p proto.Message) error {
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewDecoder(f).Decode(p)
}
