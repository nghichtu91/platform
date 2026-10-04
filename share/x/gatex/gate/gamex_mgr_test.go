package gate

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nghichtu91/platform/share/planx/etcd"
	discovery "github.com/nghichtu91/platform/share/planx/etcd_ser_discovery"
	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"
)

func init() {
	zaplog.DebugConsoleEncode()
	zaplog.InitZapLog("", nil, "")
}

func TestAddService(t *testing.T) {
	var mgr = NewGameMgr(NewGateServer(nil))

	type TestCase struct {
		SerId                string
		LogicSid, PhysicsSid uint
		Addr                 string
	}

	var toServiceInfo = func(tc *TestCase) discovery.ServiceInfo {
		var buf [4]byte
		_, err := rand.Read(buf[:])
		require.NoErrorf(t, err, "rand.Read err")
		tc.Addr = fmt.Sprintf("%x", buf[:])
		bs, _ := json.Marshal(discovery.GamexExtra{
			PhysicsSid: tc.PhysicsSid,
		})
		return discovery.ServiceInfo{
			SerId: discovery.ServiceId{
				SerTyp: discovery.ServiceType{
					Gid:    "0",
					SerTyp: etcd.Ser_Gamex,
				},
				SerId: tc.SerId,
			},
			PrivateAddr: tc.Addr,
			ExtraInfo:   string(bs),
		}
	}

	var serIds = []*TestCase{
		{LogicSid: 1, PhysicsSid: 1},
		{LogicSid: 2, PhysicsSid: 1},
		{LogicSid: 3, PhysicsSid: 1},
		{LogicSid: 4, PhysicsSid: 1},
		{LogicSid: 5, PhysicsSid: 8},
		{LogicSid: 6, PhysicsSid: 8},
		{LogicSid: 7, PhysicsSid: 8},
		{LogicSid: 8, PhysicsSid: 8},
		{LogicSid: 9},
		{LogicSid: 10},
		{LogicSid: 5, PhysicsSid: 1},
	}
	var uniqueSid = make(map[uint]struct{})
	for _, tc := range serIds {
		var strId = strconv.Itoa(int(tc.LogicSid))
		tc.SerId = strId
		mgr.OnAddService(toServiceInfo(tc))
		info := mgr.games[tc.LogicSid]
		assert.NotNil(t, info)
		info.connsForClient = append(info.connsForClient, &gpr_conn{})
	}

	for _, tc := range serIds {
		logicInfo, logicOk := mgr.games[tc.LogicSid]
		physicsInfo, physicsOk := mgr.games[logicInfo.getPhysicsSid()]
		require.NotNilf(t, logicInfo, "logicInfo %d", tc.LogicSid)
		require.Truef(t, logicOk, "logicOk %d", tc.LogicSid)
		require.NotNilf(t, physicsInfo, "physicsInfo %d", tc.PhysicsSid)
		require.Truef(t, physicsOk, "physicsOk %d", tc.PhysicsSid)

		conn, err := mgr.getConn("0:" + tc.SerId + ":0")
		require.NoError(t, err)
		require.Equal(t, conn, physicsInfo.getConn())

		uniqueSid[tc.LogicSid] = struct{}{}
	}

	assert.Equal(t, len(uniqueSid), len(mgr.games))

	{

		var firstTc = serIds[0]
		var secondTc = serIds[1]

		mgr.games[firstTc.LogicSid].connsForClient = nil
		mgr.games[secondTc.LogicSid].connsForClient = nil
		mgr.OnDelService(toServiceInfo(firstTc).SerId)

		conn1, err1 := mgr.getConn("0:" + firstTc.SerId + ":0")
		conn2, err2 := mgr.getConn("0:" + secondTc.SerId + ":0")
		require.Nil(t, conn1)
		require.Nil(t, conn2)
		require.Error(t, err1)
		require.Error(t, err2)

		t.Logf("firstTc err %v, secondTc err %v", err1, err2)

	}

	for _, info := range mgr.games {
		info.connsForClient = nil
	}

	{
		// 测试停止时的清理
		mgr.discoveryMgr = discovery.NewDiscoveryMgr(discovery.DiscoveryConfig{
			EtcdRoot: "",
			BeDiscoverySerTyp: discovery.ServiceType{
				Gid:    fmt.Sprintf("%d", 0),
				SerTyp: etcd.Ser_Gamex,
			},
			HasPrivateAddr: true,
		}, mgr, nil)

		mgr.Stop()
	}
}
