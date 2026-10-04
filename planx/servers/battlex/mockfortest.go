package battlex

import (
	"github.com/nghichtu91/platform/share/planx/servers/planxprotogen"
)

func StartBattleMgrForTest() {
	gBattleServsMgr = &battleServsMgr{
		serverId: "fortest",
	}
}

type mockSender struct {
}

func (s *mockSender) send(msg *planxprotogen.BattleMsg) bool {
	return true
}
