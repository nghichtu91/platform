package handles

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/opentracing/opentracing-go"
	"github.com/nghichtu91/platform/share/planx/3rd_party_api/yidun"
	log "github.com/nghichtu91/platform/share/planx/tilogs"
	"github.com/nghichtu91/platform/share/planx/tilogs/chatlog"
	"github.com/nghichtu91/platform/share/planx/timeutil"
	"github.com/nghichtu91/platform/share/x/chat/api/nats_cb"
	pb "github.com/nghichtu91/platform/share/x/chat/api/protogen"
	conf "github.com/nghichtu91/platform/share/x/chat/logicx/config"
	"github.com/nghichtu91/platform/share/x/chat/logicx/logiclog"
	"github.com/nghichtu91/platform/share/x/common/allmetrics"
)

type PushMsgHandle struct {
	msg *pb.GamexChatMsg
	Cb  nats_cb.LogicNatsCallBack
}

func (req *PushMsgHandle) NewProtoMsg() proto.Message {
	req.msg = chatMsgPool.Get().(*pb.GamexChatMsg)
	return req.msg
}

func (req *PushMsgHandle) GetMsg() proto.Message {
	return req.msg
}

func (req *PushMsgHandle) Handle(fatherSpan opentracing.Span) proto.Message {
	defer func() {
		req.msg.Reset()
		chatMsgPool.Put(req.msg)
		req.msg = nil
	}()

	log.L().Infof("logic server nats receive push msg:%v", req.msg)

	if fatherSpan != nil {
		span := opentracing.GlobalTracer().StartSpan("GameChatMsg.logic", opentracing.ChildOf(fatherSpan.Context()))
		defer span.Finish()
	}
	strs := strings.Split(req.msg.GetTargetId(), "_")
	if len(strs) > 0 {
		allmetrics.LogicCountAddNewRequest(strs[0])
	}
	ret := req.Cb.Push(fatherSpan, req.msg)
	return ret
}

// TextValidHandler 用于验证并上报非聊天的用户生成内容，譬如玩家昵称、军团名称、军团宣言等
// 会连接易盾和英雄等三方API，为了不影响游戏内体验，连接超时或遇到错误会放行
type TextValidHandler struct {
	msg *pb.TextValidateReq
}

func (req *TextValidHandler) NewProtoMsg() proto.Message {
	req.msg = validMsgPool.Get().(*pb.TextValidateReq)
	return req.msg
}

func (req *TextValidHandler) GetMsg() proto.Message {
	return req.msg
}

func (req *TextValidHandler) Handle(fatherSpan opentracing.Span) proto.Message {
	resp := new(pb.TextValidateResp)
	defer func() {
		req.msg.Reset()
		validMsgPool.Put(req.msg)
		req.msg = nil
	}()

	// 因为IsOther被设置成了required，初始值得给一个赋值
	resp.IsOther = proto.Bool(false)

	log.L().Infof("logic server nats receive text valid msg:%v", req.msg)

	if fatherSpan != nil {
		span := opentracing.GlobalTracer().StartSpan("TextValidHandler.logic", opentracing.ChildOf(fatherSpan.Context()))
		defer span.Finish()
	}

	ret, labels, err := yidun.Check(req.msg, fatherSpan)

	var result int32
	if err == nil {
		result = int32(ret)
	} else {
		// 超时埋点
		chatlog.LogChatPlatform(
			logiclog.CommonInfo(
				req.msg.GetAcid(), // operator
				chatlog.YiDunTimeOut,
				false,             // 是否是房间消息
				false,             // 是否是喇叭消息
				req.msg.GetAcid(), // sender
				req.msg.GetNickname(),
				req.msg.GetReceiver(),
				req.msg.GetDeviceID(), // device
				req.msg.GetText(),     // 文本内容
				timeutil.Now().Unix()))
		log.L().Errorf("response from yiDun failed, err %s", err.Error())
		result = int32(yidun.RetPass)
	}
	willSend := true
	isOther := false
	isPolitcs := false
	// 判断是否色情、涉政，此类消息不向飞书群推送
	for _, ele := range labels {
		switch ele.Lable {
		case 100: // 色情
			if !timeutil.IsAsiaShanghaiTZ() {
				willSend = false
			}
		case 500: // 涉政
			if ele.Level == 1 || ele.Level == 2 { // 涉政嫌疑级别也拦下
				result = int32(yidun.RetFail)
				isPolitcs = true
			}
		}
	}
	// 不通过发送到群机器人
	if result == yidun.RetFail && willSend {
		if conf.Cfg.FeiShuRobot.FeiShuEnable {
			if labels == nil {
				return resp
			}
			timeStamp := time.Now().Unix()
			sign, err := genSign(conf.Cfg.FeiShuRobot.FeiShuSercetKey, timeStamp)
			if err != nil {
				return resp
			}
			// 充值
			score := req.msg.GetScore()
			var reason string
			for _, ele := range labels {
				if ele == nil {
					continue
				}
				// 屏蔽类型是否是其他
				if ele.Lable == 900 {
					isOther = true
				}
				desc, _ := conf.LabelCode2Desc[ele.Lable]
				levelDesc, _ := conf.LabelCode2Desc[ele.Level]
				reason += desc + "(" + levelDesc + ") "
			}
			text := fmt.Sprintf(" 文本内容:%s\\n 用户id: %s\\n 昵称:%s\\n IP地址:%s\\n  操作系统:%s \\n 命中类型:%s \\n 累充积分:%d",
				req.msg.GetText(), req.msg.GetAcid(), req.msg.GetNickname(), req.msg.GetIp(), req.msg.GetDeviceOS(), reason, score)
			// 积分大于配置，飞书发送消息后直接通过
			if score >= conf.Cfg.YiDun.Score && !isOther && !isPolitcs {
				result = yidun.RetPass
				text += "(未阻拦)"
			} else {
				text += "(已阻拦)"
			}
			sendData := `{
			"msg_type": "text",
			"timestamp": ` + strconv.FormatInt(timeStamp, 10) + `,
			"sign": "+` + sign + `",
			"content": {"text": "` + "消息通知\\n" + text + `"}
			}`

			var url string
			if !timeutil.IsAsiaShanghaiTZ() {
				url = conf.Cfg.FeiShuRobot.FeiShuUrl
			} else {
				for _, ele := range labels {
					switch ele.Lable {
					case 100:
						url = conf.Cfg.FeiShuRobot.FeiShuCn100Url
					case 200:
						url = conf.Cfg.FeiShuRobot.FeiShuCn200Url
					case 500:
						url = conf.Cfg.FeiShuRobot.FeiShuCn500Url
					default:
						url = conf.Cfg.FeiShuRobot.FeiShuCnUrl
					}
				}
			}

			contexType := conf.Cfg.FeiShuRobot.FeiShuCtxType
			client := http.Client{Timeout: time.Millisecond * 500}
			response, _ := client.Post(url, contexType, strings.NewReader(sendData))

			if response != nil {
				defer response.Body.Close()
			}
		}
	}

	resp.NsRet = proto.Int32(result)
	resp.IsOther = proto.Bool(isOther)
	return resp
}

func genSign(secret string, timestamp int64) (string, error) {
	// timestamp + key 做sha256, 再进行base64 encode
	stringToSign := fmt.Sprintf("%v", timestamp) + "\n" + secret
	var data []byte
	h := hmac.New(sha256.New, []byte(stringToSign))
	_, err := h.Write(data)
	if err != nil {
		return "", err
	}
	sign := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return sign, nil
}
