package scene

import (
	"context"
	"fmt"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/opentracing/opentracing-go"
	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/nats_cli"
	"github.com/nghichtu91/platform/share/planx/servers/planxprotogen"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

func (dm *dynamicNatsMgr) RegHandlers() {
	tilogs.L().Infof("dynamicNatsMgr RegHandlers %v %v %v %v %v %v",
		dm.gid, dm.uID, nats_cli.SceneModuleGroup(strconv.Itoa(int(dm.gid)), DynamicSceneModule, fmt.Sprintf("%d", dm.uID)))
	nats_cli.GetHandlersMgr().RegWithLock(nats_cli.SceneModuleGroup(strconv.Itoa(int(dm.gid)), DynamicSceneModule, fmt.Sprintf("%d", dm.uID)),
		&DynamicSceneMgr{mgr: dm})

}

func (dm *dynamicNatsMgr) UnRegHandlers() {
	tilogs.L().Infof("dynamicNatsMgr UnRegHandlers %v %v %v %v %v %v",
		dm.gid, dm.uID, nats_cli.SceneModuleGroup(strconv.Itoa(int(dm.gid)), DynamicSceneModule, fmt.Sprintf("%d", dm.uID)))
	if err := nats_cli.GetHandlersMgr().UnRegWithLock(nats_cli.SceneModuleGroup(strconv.Itoa(int(dm.gid)),
		DynamicSceneModule, fmt.Sprintf("%d", dm.uID))); err != nil {
		tilogs.L().Errorf("dynamicNatsMgr UnRegHandlers err %v", err)
	}

}

type iCmd interface {
	run(m *dynamicNatsMgr)
}

// SceneCmd Scene的Nats处理
type DynamicNatsCmd struct {
	info *planxprotogen.DynamicSceneCmd
	ret  chan *planxprotogen.DynamicSceneRet
}

type DynamicSceneMgr struct {
	mgr *dynamicNatsMgr
	msg *planxprotogen.DynamicSceneCmd
}

func (req *DynamicSceneMgr) NewProtoMsg() proto.Message {
	req.msg = &planxprotogen.DynamicSceneCmd{}
	return req.msg
}

func (req *DynamicSceneMgr) GetMsg() proto.Message {
	return req.msg
}

func (req *DynamicSceneMgr) Handle(opentracing.Span) proto.Message {
	tilogs.L().Debugf("DynamicSceneMgr Handle req: %s", req.msg.String())

	cmd := &DynamicNatsCmd{
		info: req.msg,
		ret:  make(chan *planxprotogen.DynamicSceneRet, 1),
	}
	if err := req.mgr.SendCmd(cmd); err != nil {
		tilogs.L().Errorf("DynamicSceneMgr sendError %s %v", err.Error(), cmd)
		ret := &planxprotogen.DynamicSceneRet{
			Code: 2,
			Msg:  err.Error(),
		}
		return ret
	}
	ctx, cancel := context.WithTimeout(context.Background(), planx.ClientTimeOut)
	defer cancel()
	select {
	case res := <-cmd.ret:
		return res
	case <-ctx.Done():
		err := fmt.Errorf("DynamicSceneMgr : Cmd ret timeout, %v", cmd)
		tilogs.L().Errorf(err.Error())
		ret := &planxprotogen.DynamicSceneRet{
			Code: 3,
			Msg:  err.Error(),
		}
		return ret
	}
}

func (dm *dynamicNatsMgr) SendCmd(c iCmd) error {
	ctx, cancel := context.WithTimeout(context.Background(), planx.ClientTimeOut)
	defer cancel()
	select {
	case dm.cmdChan <- c:
	case <-ctx.Done():
		err := fmt.Errorf("%w: DynamicSceneMgr Cmd timeout, %v", planx.ErrSendFail, c)
		return err
	}
	return nil
}

func (ds *DynamicNatsCmd) run(dm *dynamicNatsMgr) {
	fatherSpan := opentracing.GlobalTracer().StartSpan(fmt.Sprintf("dynamicNatsMgr-%v", ds.info.GetTyp()))
	defer fatherSpan.Finish()

	tilogs.L().Infof("dynamicNatsMgr GetCommand type:%v uId:%v", ds.info.GetTyp(), dm.uID)
	ret := &planxprotogen.DynamicSceneRet{
		Code: 0,
		Msg:  "success",
	}
	switch ds.info.GetTyp() {
	case planxprotogen.DynamicSceneCmdTyp_SLNone:
		break
	case planxprotogen.DynamicSceneCmdTyp_SLGetPlayerPosition:
		ret = dm.dynamicNatsHandle(ds.info.GetMsg())
	case planxprotogen.DynamicSceneCmdTyp_SLAsyncMsg:
		dm.asyncDynamicNatsHandle(ds.info.GetMsg())
		ret = nil
	default:
		tilogs.L().Errorf("DynamicSceneMgr request function type notfound %v", ds.info.GetTyp())
	}
	ds.ret <- ret
}
