package battle_check

import (
	"fmt"
	"testing"
	"time"

	"github.com/nghichtu91/platform/share/planx/signalhandler"
	"github.com/nghichtu91/platform/share/planx/util"
)

func TestFrameDataValidator_Start(t *testing.T) {
	waitGroup := &util.WaitGroupWrapper{}
	//handle kill signal
	waitGroup.Wrap(func() { signalhandler.SignalKillHandle() })
	v := GenFrameDataValidator(1, 1, 1)
	v.Start()
	//创建定时器并设置定时时间
	ticker := time.NewTicker(1 * time.Second)
	waitGroup.Wrap(func() {
		defer func() {
			v.End()
			waitGroup.Done()
		}()
		var frame = 0
		for {
			frame += 1
			if frame >= 15 {
				return
			}
			//监听定时器
			select {
			case <-ticker.C:
				sendClientFrameData(v, frame)
			}
		}
	})
	waitGroup.Wait()
}

func sendClientFrameData(v *FrameDataValidator, frameid int) {
	accid1 := "accid1"
	data1 := FramValidateData{}
	data1.result = []byte(genStringByFrameId(frameid, accid1))
	data1.resultContext = ""
	data1.accid = accid1
	data1.frameID = int16(frameid)
	v.write(data1)

	accid2 := "accid2"
	data2 := FramValidateData{}
	data2.result = []byte(genStringByFrameId(frameid, accid2))
	data2.resultContext = ""
	data2.accid = accid2
	data2.frameID = int16(frameid)
	v.write(data2)
}

func genStringByFrameId(frameid int, accid string) string {
	if frameid < 10 {
		return "fram id " + fmt.Sprintf("%d", frameid)
	} else {
		return "fram id " + fmt.Sprintf("%d", frameid) + accid
	}
}
