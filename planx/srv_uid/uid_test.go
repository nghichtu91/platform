package srv_uid

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/nghichtu91/platform/share/planx/etcd"
)

func TestGetSrvUID(t *testing.T) {
	tests := []struct {
		typ string
		id  string
		out uint16
	}{
		{"", "", 0},
		{"", " ", 0},
		{etcd.ServerSer_GMServer, "gmserver", UIDGMServer},
		{etcd.Server_Gamex, "gamex160001", 1},
		{etcd.Server_Crossx, "cross0", UIDCrossxBegin},
		{etcd.Server_Crossx, "cross1", UIDCrossxBegin + 1},
		{etcd.Server_Crossx, "cross00", UIDCrossxBegin},
		{etcd.Server_Crossx, "cross01", UIDCrossxBegin + 1},
		{etcd.Server_Battle, "test111plus", UIDBattlexBegin + 111},
		{etcd.Server_Scene, "test111plus222", UIDSceneBegin + 111},
		{etcd.ServerXScene, "xscene1600001", UIDXSceneBegin + 1},
		{etcd.Server_Stressx, "极无双2", UIDStressxBegin + 2},
	}

	for i, test := range tests {
		assert.Equal(t, test.out, GetSrvUID(test.typ, test.id), i)
	}
}
