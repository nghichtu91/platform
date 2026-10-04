package scene

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/x/common/msg/scene/pb"

	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/servers/planxprotogen"
)

func (dm *dynamicNatsMgr) dynamicNatsHandle(msg []byte) *planxprotogen.DynamicSceneRet {
	tilogs.L().Debugf("Start Handle dynamicNatsHandle")
	ret := &planxprotogen.DynamicSceneRet{
		Code: 0,
		Msg:  "",
	}

	ctx, cancel := context.WithTimeout(context.Background(), planx.InnerTimeOut)
	defer cancel()
	Msg := &pb.SceneMsg{}
	err := proto.Unmarshal(msg, Msg)
	if err != nil {
		tilogs.L().Errorf("[%v] dynamicNatsHandle %v %v", dm.uID, err)
		return ret
	}

	for _, v := range dm.sUC {
		if v == nil {
			tilogs.L().Infof("[%v] dynamicNatsHandle GetSceneFailed %v", dm.uID, v)
			continue
		}

		resp := v.inSync(ctx, Msg)
		if resp.msg == nil {
			//tilogs.L().Errorf("[%v] dynamicNatsHandle receive is nil", dm.uID)
			continue
		}

		ret.RespInfo = append(ret.RespInfo, resp.msg.GetMsgData())
	}
	if len(ret.RespInfo) <= 0 {
		return nil
	}
	return ret
}

func (dm *dynamicNatsMgr) asyncDynamicNatsHandle(msg []byte) {
	tilogs.L().Debugf("Start Handle dynamicNatsHandle")
	ctx, cancel := context.WithTimeout(context.Background(), planx.InnerTimeOut)
	defer cancel()
	Msg := &pb.SceneMsg{}
	err := proto.Unmarshal(msg, Msg)
	if err != nil {
		tilogs.L().Errorf("[%v] dynamicNatsHandle %v %v", dm.uID, err)
		return
	}

	for _, v := range dm.sUC {
		if v == nil {
			tilogs.L().Infof("[%v] dynamicNatsHandle GetSceneFailed %v", dm.uID, v)
			continue
		}
		v.inAsync(ctx, Msg)
	}

	return
}
