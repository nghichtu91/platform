package yidun

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/astaxie/beego/httplib"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/nghichtu91/platform/share/planx/rand_pool"
	"github.com/nghichtu91/platform/share/planx/tilogs"
)

// CommonReq 易盾请求通用字段
type CommonReq struct {
	// SecretId 产品秘钥 id ，由易盾内容安全服务分配，产品标识
	SecretId string `form:"secretId" json:"secretId"`

	// BusinessId 推荐使用businessId(业务id)鉴权，使用无限制且灵活
	BusinessId string `form:"businessId" json:"businessId"`

	// Timestamp 请求当前 UNIX 时间戳，请注意服务器时间是否同步
	Timestamp string `form:"timestamp" json:"timestamp"`

	// Nonce 随机整数，与 timestamp 联合起来，用于防止重放攻击
	// 最大长度11
	Nonce string `form:"nonce" json:"nonce"`

	// SignatureMethod
	// 指定签名算法，国密：SM3,如果使用md5默认签名算法，可以不传
	// SignatureMethod string `form:"signatureMethod" json:"signatureMethod,omitempty"`

	// Version 版本号，可选
	Version string `form:"version" json:"version,omitempty"`

	// Signature 签名
	// 这里json加omitempty标签是为了排序时能跳过这个字段
	Signature string `form:"signature" json:"signature,omitempty"`
}

// CommonResp 易盾回复通用字段
type CommonResp struct {
	// Code
	// 接口调用状态，200:正常，其他值：调用出错
	Code int `json:"code"`

	// Msg
	// 结果说明，如果接口调用出错，那么返回错误描述，成功返回 ok
	Msg string `json:"msg"`
}

type textValidReq struct {
	CommonReq

	// DataID
	// 最大长度128
	// 数据唯一标识，能够根据该值定位到该条数据，如对数据结果有异议，可以发送该值给客户经理查询
	// 这里使用uuid做编号
	DataID string `form:"dataId" json:"dataId"`

	// Content
	// 最大长度100000
	// 用户发表内容，建议对内容中json、表情符、HTML标签、UBB标签等做过滤，只传递纯文本，以减少误判概率
	Content string `form:"content" json:"content"`

	// Version
	// 接口版本号，可选值 v5.1
	Version string `form:"version"`

	// ReceiveUid
	// 接受消息的用户标识，私聊/评论回复场景使用，易盾可根据该id关联检测，辅助机审策略精准调优
	ReceiveUid string `form:"receiveUid" json:"receiveUid,omitempty"`

	// Category
	// 最大长度128
	// 来源，用于展示渠道名称，应用名称等
	// Category string `form:"category" json:"category,omitempty"`

	// Account
	// 最大长度128
	// 用户唯一标识，与易盾账号画像库匹配，建议抄送，辅助机审策略精准调优
	Account string `form:"account" json:"account,omitempty"`

	// Nickname
	// 最大长度128
	// 用户昵称，建议抄送，辅助机审策略精准调优
	NickName string `form:"nickname" json:"nickname,omitempty"`

	// Level
	// 用户等级，0未知，1初级，2中级，3高级
	// Level int `form:"level" json:"level,omitempty"`

	// IP
	// 最大长度128
	// 用户IP地址，建议抄送，辅助机审策略精准调优
	IP string `form:"ip" json:"ip,omitempty"`

	// DeviceId
	// 最大长度128
	// 用户设备 id，与易盾设备画像库匹配，明文请转大写传入；MD5加密请明文转大写后MD5计算，再转大写传入，建议抄送
	DeviceId string `form:"deviceId" json:"deviceId,omitempty"`

	// extStr1
	// 自定义参数1，用于传递中台ID
	ExtStr1 string `form:"extStr1" json:"extStr1,omitempty"`

	// extStr2
	// 自定义参数2，用于传递操作系统
	ExtStr2 string `form:"extStr2" json:"extStr2,omitempty"`

	// extLon1
	// 自定义参数3，用于传输消息类型（1文字，2语音）
	ExtLon1 uint64 `form:"extLon1" json:"extLon1,omitempty"`
}

type textValidResp struct {
	CommonResp

	Result struct {
		AntiSpam struct {
			// TaskID 任务ID
			// 示例值："fx6sxdcd89fvbvg4967b4787d78a"
			TaskID string `json:"taskId"`

			// DataID 数据ID
			// 与发送过去的ID一致
			DataID string `json:"dataId"`

			// Suggestion
			// 建议动作，0：通过，1：嫌疑，2：不通过
			Suggestion int `json:"suggestion"`
			// labels,命中原因
			Labels []Lable `json:"labels"`
		} `json:"antispam"`
	} `json:"result"`
}

type Lable struct {
	Lable int64 `json:"label"` // 命中分类，分类信息，100：色情，200：广告，260：广告法，300：暴恐，400：违禁，500：涉政，600：谩骂，700：灌水，900：其他，1100：涉价值观
	Level int64 `json:"level"` // 命中级别，示例值：1：嫌疑，2：不通过
}

// NewTextValidReq 生成新文本检测请求
func NewTextValidReq() *textValidReq {
	return &textValidReq{
		CommonReq: newCommonReq(),
		DataID:    uuid.New().String(),
		Version:   cfg.ver,
	}
}

// newCommonReq 生成不带签名的请求公共参数
func newCommonReq() CommonReq {
	return CommonReq{
		SecretId:   cfg.secretID,
		BusinessId: cfg.businessId,
		Timestamp:  strconv.Itoa(int(time.Now().Unix())),
		Nonce:      strconv.Itoa(int(rand_pool.Int31())),
	}
}

// reset 重置内容，避免对象池被污染
func (req *textValidReq) reset() {
	req.DataID = uuid.New().String()
	req.Timestamp = strconv.Itoa(int(time.Now().Unix()))
	req.Nonce = strconv.Itoa(int(rand_pool.Int31()))
}

// handle 逻辑处理
func (req *textValidReq) handle(span opentracing.Span) (int, []*Lable, error) {
	var (
		err error
	)

	if span != nil {
		childSpan := opentracing.GlobalTracer().StartSpan("TextValidReq.http", opentracing.ChildOf(span.Context()))
		defer childSpan.Finish()
	}

	st := time.Now()

	resp := getTVResp()
	defer putTVResp(resp)

	// 所有初始字段放入map[string]string结构
	jsonBytes, _ := json.Marshal(req)
	params := make(map[string]interface{}, 16)
	if err = json.Unmarshal(jsonBytes, &params); err != nil {
		return RetPass, nil, err
	}

	// http请求
	hReq := httplib.Post(cfg.url).
		Header(headerTypeKey, headerTypeValue).
		SetTimeout(HttpTimeOut, HttpTimeOut).
		Retries(Retries)

	// 排序，顺便将param加入请求
	keys := make([]string, 0, len(params))
	for k, v := range params {
		keys = append(keys, k)
		switch reflect.TypeOf(v).Kind() {
		case reflect.String:
			hReq.Param(k, v.(string))
		case reflect.Float64:
			hReq.Param(k, strconv.FormatFloat(v.(float64), 'f', -1, 64))
		default:
			tilogs.L().Warnf("invalid param key %v value %v", k, v)
		}
	}
	sort.Strings(keys)

	// 生成字符串
	buff := bytes.NewBuffer(make([]byte, 0, 1024))
	for _, key := range keys {
		buff.WriteString(key)
		switch reflect.TypeOf(params[key]).Kind() {
		case reflect.String:
			buff.WriteString(params[key].(string))
		case reflect.Float64:
			buff.WriteString(strconv.FormatFloat(params[key].(float64), 'f', -1, 64))
		}
	}
	buff.WriteString(cfg.secretKey)

	// 生成md5
	h := md5.New()
	h.Write(buff.Bytes())
	sign := hex.EncodeToString(h.Sum(nil))

	// http参数加入签名
	hReq.Param(signKey, sign)
	if err = hReq.ToJSON(resp); err != nil {
		return RetPass, nil, err
	}

	if resp.Code != http.StatusOK {
		return RetPass, nil, fmt.Errorf(resp.Msg)
	}

	// 超过3秒的请求，打印resp看结果
	if time.Now().Sub(st) > 3*time.Second {
		tilogs.L().Errorf("textValidReq handle more than 3 seconds, resp %+v", resp)
	}

	// labels赋值
	labels := make([]*Lable, 0, len(resp.Result.AntiSpam.Labels))
	for _, lable := range resp.Result.AntiSpam.Labels {
		labels = append(labels, &Lable{Lable: lable.Lable, Level: lable.Level})
	}

	return resp.Result.AntiSpam.Suggestion, labels, nil
}
