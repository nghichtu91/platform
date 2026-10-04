package scene

import (
	"sync"

	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/util"
)

type dynamicNatsMgr struct {
	*sceneMgr

	gid uint
	uID int64
	sUC []*scene // sceneUnderControl scene UniqId

	wait util.WaitGroupWrapper
	once sync.Once

	quit    chan bool
	cmdChan chan iCmd
}

// start natsCMD启动。
func (dm *dynamicNatsMgr) start() {
	tilogs.L().Infof("start listen dynamicScene nats uId %v", dm.uID)
	dm.RegHandlers()
	dm.wait.Wrap(func() {
		defer func() {
			tilogs.L().Infof("dynamicScene mapId: %v uId %v stop", dm.uID)
		}()
		for {
			select {
			case _c := <-dm.cmdChan:
				func(c iCmd) {
					defer tilogs.PanicCatcher("dynamicScene panic, cmd %v", c)
					c.run(dm)
				}(_c)
			case <-dm.quit:
				tilogs.L().Infof("dynamicNatsMgr receive quit uID: %v", dm.uID)
				return
			}
		}
	})
}

func (dm *dynamicNatsMgr) stop() {
	dm.once.Do(func() {
		dm.UnRegHandlers()
		close(dm.quit)
		dm.wait.Wait()
	})
}
