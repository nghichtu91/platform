package silence_sys

import (
	"context"
	"strconv"
	"sync"

	"github.com/opentracing/opentracing-go"

	"github.com/nghichtu91/platform/share/planx/silence_sys/planxprotogen"

	"github.com/golang/protobuf/proto"
	"github.com/nghichtu91/platform/share/planx/nats_cli"

	"github.com/nghichtu91/platform/share/planx"
	"github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/timeutil"

	"github.com/nghichtu91/platform/share/planx/util"
)

type ICmd interface {
	run(m *SilenceSys)
}

type SilenceSys struct {
	SilenceSysCache map[string]int64 // 玩家禁言缓存<acid, endTime>
	cmdChan         chan ICmd
	ShardId         uint32
	Gid             uint32
	once            sync.Once
	quit            chan struct{}
	loadSuccess     bool
}

func NewSilenceSysManager(gid uint32, shardId uint32) *SilenceSys {
	s := &SilenceSys{
		SilenceSysCache: make(map[string]int64, 0),
		cmdChan:         make(chan ICmd, 1024),
		Gid:             gid,
		ShardId:         shardId,
		loadSuccess:     false,
		quit:            make(chan struct{}, 1),
	}
	s.regRecSilenceSysHandler()
	return s
}

func (s *SilenceSys) Start(w *util.WaitGroupWrapper) error {
	s.updateSilenceSys()

	w.Wrap(func() {
		defer func() {
			if err := recover(); err != nil {
				tilogs.L().Errorf("SilenceSys.start panic err:%v", err)
			}
			tilogs.L().Infof("SilenceSys.start close")
		}()

		loadGMSilenceSysTimer := timeutil.TimerSec.After(planx.LoadGMSilenceSys)
		for {
			select {
			case <-s.quit:
				return
			case _c := <-s.cmdChan:
				func(c ICmd) {
					defer tilogs.PanicCatcher("SilenceSys panic, cmd %v", c)
					c.run(s)
				}(_c)
			case <-loadGMSilenceSysTimer:
				func() {
					defer tilogs.PanicCatcher("SilenceSys loadGMSilenceSysTimer panic")
					if s.loadSuccess {
						loadGMSilenceSysTimer = nil
					} else {
						s.updateSilenceSys()
						loadGMSilenceSysTimer = timeutil.TimerSec.After(planx.LoadGMGlobalMail)
					}
				}()
			}
		}
	})

	return nil
}

func (s *SilenceSys) AfterStart() {

}

func (s *SilenceSys) BeforeStop() {

}

func (s *SilenceSys) Stop() {
	s.once.Do(func() {
		close(s.quit)
	})
}

type IsPlayerSilenceSys struct {
	acid string
	ret  chan bool
}

func (cmd *IsPlayerSilenceSys) run(sys *SilenceSys) {
	time, ok := sys.SilenceSysCache[cmd.acid]
	if !ok {
		cmd.ret <- false
		return
	}

	if time > timeutil.Now().Unix() || time == 0 {
		cmd.ret <- true
		return
	}

	cmd.ret <- false
}

func (s *SilenceSys) IsPlayerSilenceSys(acid string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), planx.InnerTimeOut)
	defer cancel()

	cmd := &IsPlayerSilenceSys{
		ret:  make(chan bool, 1),
		acid: acid,
	}
	select {
	case s.cmdChan <- cmd:
	case <-ctx.Done():
		tilogs.L().Errorf("%s IsPlayerSilenceSys timeout", acid)
		return false
	}

	isSilence := false
	select {
	case isSilence = <-cmd.ret:
		return isSilence
	case <-ctx.Done():
		tilogs.L().Errorf("IsPlayerSilenceSys <-cmd.ret timeout, err: %v, cmd: %v", planx.ErrSendFail, cmd)
		return isSilence
	}
}

type GetPlayerSilenceSys struct {
	his map[string]string
	ret chan map[string]string
}

func (cmd *GetPlayerSilenceSys) run(sys *SilenceSys) {
	result := make(map[string]string, len(cmd.his))
	for uuid, fromId := range cmd.his {
		if time, ok := sys.SilenceSysCache[fromId]; ok {
			if time == 0 || time > timeutil.Now().Unix() {
				continue
			}
		}

		result[uuid] = fromId
	}

	cmd.ret <- result
}

func (s *SilenceSys) GetPlayerSilenceSys(his map[string]string) map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), planx.InnerTimeOut)
	defer cancel()

	cmd := &GetPlayerSilenceSys{
		his: his,
		ret: make(chan map[string]string, 1),
	}
	select {
	case s.cmdChan <- cmd:
	case <-ctx.Done():
		tilogs.L().Errorf("%s GetPlayerSilenceSys timeout")
		return nil
	}

	silenceInfo := make(map[string]string, 0)
	select {
	case silenceInfo = <-cmd.ret:
		return silenceInfo
	case <-ctx.Done():
		tilogs.L().Errorf("GetPlayerSilenceSys <-cmd.ret timeout, err: %v, cmd: %v", planx.ErrSendFail, cmd)
		return silenceInfo
	}
}

type UpdatePlayerSilenceSys struct {
	acid    string
	endTime int64
	isBan   bool
}

func (cmd *UpdatePlayerSilenceSys) run(sys *SilenceSys) {
	if cmd.isBan {
		sys.SilenceSysCache[cmd.acid] = cmd.endTime
	} else {
		if _, ok := sys.SilenceSysCache[cmd.acid]; ok {
			delete(sys.SilenceSysCache, cmd.acid)
		}
	}
}

func (s *SilenceSys) UpdatePlayerSilenceSys(acid string, endTime int64, isban bool) {
	ctx, cancel := context.WithTimeout(context.Background(), planx.InnerTimeOut)
	defer cancel()

	cmd := &UpdatePlayerSilenceSys{
		acid:    acid,
		endTime: endTime,
		isBan:   isban,
	}
	select {
	case s.cmdChan <- cmd:
	case <-ctx.Done():
		tilogs.L().Errorf("%s UpdatePlayerSilenceSys timeout")
		return
	}
}

func (s *SilenceSys) updateSilenceSys() {
	silenceSysData := s.getSilenceSysInfoFromGM()
	if silenceSysData == nil {
		return
	}

	s.SilenceSysCache = silenceSysData
}

func (s *SilenceSys) getSilenceSysInfoFromGM() map[string]int64 {
	if s.Gid == 0 {
		tilogs.L().Errorf("<func getSilenceSysInfoFromGM(), Gid is 0")
		return nil
	}
	resp := &planxprotogen.GetSilenceSysInfoResp{}
	gmTopic := nats_cli.GMGidSubj(strconv.Itoa(int(s.Gid)))
	err := nats_cli.RequestResponseWithTimeout(planx.UpTimeOut*2, gmTopic, &planxprotogen.GetSilenceSysInfoReq{
		ShardId: s.ShardId,
	}, resp)
	//raw, err := nats_cli.Request(gmTopic, &planxprotogen.GetSilenceSysInfoReq{
	//	ShardId: s.ShardId,
	//})
	if err != nil {
		tilogs.L().Errorf("<func getSilenceSysInfoFromGM(), request gm error, err is: %s>", err.Error())
		return nil
	}

	//err = proto.Unmarshal(raw, resp)
	//if err != nil {
	//	tilogs.L().Errorf("<func getSilenceSysInfoFromGM(),Unmarshal error, err is: %s>", err.Error())
	//	return nil
	//}
	if resp.GetErr() != "" {
		tilogs.L().Errorf("<func getSilenceSysInfoFromGM(),gms send error, err is: %s>", resp.GetErr())
		return nil
	}

	s.loadSuccess = true

	return resp.GetSilenceSysInfos()
}

func (s *SilenceSys) regRecSilenceSysHandler() {
	nats_cli.GetHandlersMgr().RegWithLock(nats_cli.GamexSerSubj(strconv.Itoa(int(s.Gid)), strconv.Itoa(int(s.ShardId))),
		&recSilenceSysMsg{mgr: s})
}

type recSilenceSysMsg struct {
	mgr *SilenceSys
	msg *planxprotogen.SendSilenceSysInfoReq
}

func (r *recSilenceSysMsg) NewProtoMsg() proto.Message {
	r.msg = &planxprotogen.SendSilenceSysInfoReq{}
	return r.msg
}

func (r *recSilenceSysMsg) GetMsg() proto.Message {
	return r.msg
}

func (r *recSilenceSysMsg) Handle(fatherSpan opentracing.Span) proto.Message {
	resp := &planxprotogen.SendSilenceSysInfoResp{
		ShardId: r.msg.GetShardId(),
	}
	tilogs.L().Infof("recSilenceSysMsg, msg:%v, shard:%d", r.msg, r.msg.GetShardId())
	if r.msg.GetAcid() == "" {
		tilogs.L().Errorf("SendSilenceSysInfoReq acid is empty")
		resp.Success = false
		return resp
	}

	r.mgr.UpdatePlayerSilenceSys(r.msg.GetAcid(), r.msg.GetEndTime(), r.msg.GetIsBan())

	resp.Success = true

	return resp
}
