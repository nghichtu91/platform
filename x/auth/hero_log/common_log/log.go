package common_log

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nghichtu91/platform/share/planx/timeutil"

	"github.com/nghichtu91/platform/share/planx/tilogs"

	sls "github.com/aliyun/aliyun-log-go-sdk"

	"github.com/golang/protobuf/proto"

	"github.com/google/uuid"
	"github.com/nghichtu91/platform/share/planx/util"
)

// LogCommonUploadInfo
// 通用上传参数
type LogCommonUploadInfo struct {
	BdcDeviceId  string `json:"bdc_device_id"` // 新版本设备号
	DeviceIdType string `json:"deviceid_type"` // 设备号型号
	DeviceKey    string `json:"device_key"`    // 设备号key
	DeviceModel  string `json:"device_model"`  // 设备型号
	IME          string `json:"ime"`           // 安卓设备IMEI标识符
	GAid         string `json:"gaid"`          // 安卓设备谷歌广告ID
	IdFa         string `json:"idfa"`          // IOS设备广告标识符
	IdFv         string `json:"idfv"`          // IOS设备供应商标识符
	AndroidId    string `json:"android_id"`    // 安卓ID
	OaId         string `json:"oaid"`          // 中广协OaId设备标识符
	SdkVersion   string `json:"sdk_version"`   // BDCSdk版本号
	BdcClientOs  string `json:"bdc_client_os"` // 客户端类型
	OldDeviceId  string `json:"old_device_id"` // 历史上次设备号
}

// 业务通用参数
type LogPlayerInfo struct {
	ClientOs         string // 	客户端类型：IOS|ANDROID|PAD|WEB
	BaseChannelId    string //	渠道号（注册渠道）,使用英雄互娱统一渠道ID配置表
	BaseAppChannelId string //	次级渠道号（运营渠道）,使用英雄互娱统一渠道ID配置表
	BaseDeviceId     string //	设备号,设备号
	UserId           string //	账号ID（聚合SDK关联ID）,聚合SDK关联ID
	OpenId           string //	渠道账号ID,渠道账号ID
	RoleId           string //	角色ID,角色ID
	RoleKey          string //	角色唯一key,全服务器必须唯一，方便合服以及IPO操作
	TransactionId    string //	事件关联ID,当一次事件触发多个日志产生时，需要由CP生成当前区服唯一ID用于关联用户的多个日志（同一行为触发的多个日志使用同一个关联ID），IPO必须 ,关联事件（获取代币
	UserIp           string //	用户IP,用户IP

	BdcDeviceId  string // 新版本设备号
	DeviceIdType string // 设备号型号
	DeviceKey    string // 设备号key
	DeviceModel  string // 设备型号
	IME          string // 安卓设备IMEI标识符
	GAid         string // 安卓设备谷歌广告ID
	IdFa         string // IOS设备广告标识符
	IdFv         string // IOS设备供应商标识符
	AndroidId    string // 安卓ID
	OaId         string // 中广协OaId设备标识符
	SdkVersion   string // BDCSdk版本号
	BdcClientOs  string // 客户端类型
	OldDeviceId  string // 历史上次设备号
}

func Log(playerInfo LogPlayerInfo, eventId string, kv []*sls.LogContent) {
	if Cfg == nil || !Cfg.IsOpen {
		return
	}

	content := buildLogContent(playerInfo, eventId, kv, 0)
	sendLog(content)
}

func sendLog(content []*sls.LogContent) {
	log := &sls.Log{
		Time:     proto.Uint32(uint32(timeutil.Now().Unix())),
		Contents: content,
	}

	select {
	case SendChan <- log:
	default:
		tilogs.L().Errorf("send log player error %v", log)
	}
}

func buildLogContent(playerInfo LogPlayerInfo, eventId string, kv []*sls.LogContent, t int64) []*sls.LogContent {
	if playerInfo.UserIp == "" {
		playerInfo.UserIp = "0.0.0.0"
	}
	if playerInfo.BaseDeviceId == "" {
		playerInfo.BaseDeviceId = "0"
	}
	if playerInfo.ClientOs == "" {
		playerInfo.ClientOs = "0"
	}
	if playerInfo.UserId == "" {
		playerInfo.UserId = "0"
	}

	content := make([]*sls.LogContent, 0)
	content = append(content, &sls.LogContent{
		Key:   proto.String(base_event_id),
		Value: proto.String(eventId),
	}, &sls.LogContent{
		Key:   proto.String(base_event_uuid),
		Value: proto.String(uuid.New().String()),
	}, &sls.LogContent{
		Key:   proto.String(base_appkey),
		Value: proto.String(Cfg.AppKey),
	}, &sls.LogContent{
		Key:   proto.String(base_platform),
		Value: proto.String(Cfg.Platform),
	}, &sls.LogContent{
		Key:   proto.String(base_client_os),
		Value: proto.String(playerInfo.ClientOs),
	}, &sls.LogContent{
		Key:   proto.String(base_server_id),
		Value: proto.String(Cfg.ServerId),
	}, &sls.LogContent{
		Key:   proto.String(base_channel_id),
		Value: proto.String(playerInfo.BaseChannelId),
	}, &sls.LogContent{
		Key:   proto.String(base_app_channel_id),
		Value: proto.String(playerInfo.BaseAppChannelId),
	}, &sls.LogContent{
		Key:   proto.String(base_device_id),
		Value: proto.String(playerInfo.BaseDeviceId),
	}, &sls.LogContent{
		Key:   proto.String(base_user_id),
		Value: proto.String(playerInfo.UserId),
	}, &sls.LogContent{
		Key:   proto.String(base_open_id),
		Value: proto.String(playerInfo.OpenId),
	}, &sls.LogContent{
		Key:   proto.String(base_role_id),
		Value: proto.String(playerInfo.RoleId), // ACID
	}, &sls.LogContent{
		Key:   proto.String(base_role_key),
		Value: proto.String(playerInfo.RoleKey),
	}, &sls.LogContent{
		Key:   proto.String(base_transaction_id),
		Value: proto.String(playerInfo.TransactionId),
	}, &sls.LogContent{
		Key:   proto.String(base_user_ip),
		Value: proto.String(playerInfo.UserIp),
	}, &sls.LogContent{
		Key:   proto.String(base_time_zone),
		Value: proto.String(Cfg.TimeZone),
	}, &sls.LogContent{
		Key:   proto.String(Common_upload_bdc_device_id),
		Value: proto.String(playerInfo.BdcDeviceId),
	}, &sls.LogContent{
		Key:   proto.String(Common_upload_device_id_type),
		Value: proto.String(playerInfo.DeviceIdType),
	}, &sls.LogContent{
		Key:   proto.String(Common_upload_device_key),
		Value: proto.String(playerInfo.DeviceKey),
	}, &sls.LogContent{
		Key:   proto.String(Common_upload_device_mode),
		Value: proto.String(playerInfo.DeviceModel),
	}, &sls.LogContent{
		Key:   proto.String(Common_upload_ime),
		Value: proto.String(playerInfo.IME),
	}, &sls.LogContent{
		Key:   proto.String(Common_upload_gaid),
		Value: proto.String(playerInfo.GAid),
	}, &sls.LogContent{
		Key:   proto.String(Common_upload_idfa),
		Value: proto.String(playerInfo.IdFa),
	}, &sls.LogContent{
		Key:   proto.String(Common_upload_idfv),
		Value: proto.String(playerInfo.IdFv),
	}, &sls.LogContent{
		Key:   proto.String(Common_upload_android_id),
		Value: proto.String(playerInfo.AndroidId),
	}, &sls.LogContent{
		Key:   proto.String(Common_upload_oaid),
		Value: proto.String(playerInfo.OaId),
	}, &sls.LogContent{
		Key:   proto.String(Common_upload_sdk_version),
		Value: proto.String(playerInfo.SdkVersion),
	}, &sls.LogContent{
		Key:   proto.String(Common_upload_bdc_client_os),
		Value: proto.String(playerInfo.BdcClientOs),
	}, &sls.LogContent{
		Key:   proto.String(Common_upload_old_device_id),
		Value: proto.String(playerInfo.OldDeviceId),
	})

	if t == int64(0) {
		content = append(content, &sls.LogContent{
			Key:   proto.String(base_buess_time),
			Value: proto.String(fmt.Sprint(timeutil.Now().Unix())),
		})
	} else {
		content = append(content, &sls.LogContent{
			Key:   proto.String(base_buess_time),
			Value: proto.String(fmt.Sprint(t)),
		})
	}

	content = append(content, kv...)
	return content
}

func send(logGroup *sls.LogGroup) error {
	// PostLogStoreLogs API Ref: https://intl.aliyun.com/help/doc-detail/29026.htm
	return Client.PutLogs(Cfg.ProjectName, Cfg.LogStoreName, logGroup)

	// if err == nil {
	//	return nil
	// } else {
	//	////handle exception here, you can add retryable erorrCode, set appropriate put_retry
	//	//if strings.Contains(err.Error(), sls.WRITE_QUOTA_EXCEED) || strings.Contains(err.Error(), sls.PROJECT_QUOTA_EXCEED) || strings.Contains(err.Error(), sls.SHARD_WRITE_QUOTA_EXCEED) {
	//	//	//mayby you should split shard
	//	//	time.Sleep(1000 * time.Millisecond)
	//	//} else if strings.Contains(err.Error(), sls.INTERNAL_SERVER_ERROR) || strings.Contains(err.Error(), sls.SERVER_BUSY) {
	//	//	time.Sleep(200 * time.Millisecond)
	//	//}
	//	return err
	// }
}

var (
	quitChan  chan bool
	wait      util.WaitGroupWrapper
	SendQueue []*sls.Log
	SendChan  chan *sls.Log
)

func start() {
	SendQueue = make([]*sls.Log, 0)
	SendChan = make(chan *sls.Log, 8192)
	quitChan = make(chan bool, 1)
	wait.Wrap(func() {

		timer := timeutil.TimerSec.After(time.Second)
		for {
			select {
			case log := <-SendChan:
				SendQueue = append(SendQueue, log)
			case <-timer:
				func() {
					defer tilogs.PanicCatcher("send hero platform")
					if len(SendQueue) == 0 {
						return
					}
					logList := SendQueue
					SendQueue = make([]*sls.Log, 0)
					if ret := sendLogList(logList); len(ret) != 0 {
						SendQueue = append(SendQueue, ret...)
					}
				}()
				timer = timeutil.TimerSec.After(time.Second)
			case <-quitChan:
				if len(SendQueue) > 0 {
					ret := sendLogList(SendQueue)
					if len(ret) > 0 {
						for _, v := range SendQueue {
							outPut, err := json.Marshal(v)
							if err != nil {
								tilogs.L().Errorf("json absent hero platform error %v", err)
							} else {
								tilogs.L().Errorf("record absent hero platform %s", string(outPut))
							}
						}
					}
				}
				return
			}
		}
	})
}

func sendLogList(logList []*sls.Log) []*sls.Log {

	tilogs.L().Debugf("send sendLogList")
	listLength := len(logList)
	count := listLength / 4096
	if listLength%4096 > 0 {
		count++
	}
	failIndex := -1
	for i := 0; i < count; i++ {
		endIndex := (i + 1) * 4096
		if (i+1)*4096 > listLength {
			endIndex = listLength
		}
		sends := logList[i*4096 : endIndex]
		nowTime := fmt.Sprint(timeutil.Now().Unix())
		nowTime2 := time.Now().Format("2006-01-02")
		for _, l := range sends {
			l.Contents = append(l.Contents, &sls.LogContent{
				Key:   proto.String(base_event_time),
				Value: proto.String(nowTime),
			}, &sls.LogContent{
				Key:   proto.String(base_event_time2),
				Value: proto.String(nowTime2),
			})
		}
		err := send(&sls.LogGroup{
			Topic:  proto.String(""),
			Source: proto.String(""),
			Logs:   sends})
		if err != nil {
			tilogs.L().Errorf("send failed roll back %d %v", len(sends), err)
			failIndex = i * 4096
			break
		} else {
			tilogs.L().Debugf("send ok %d", len(sends))
		}
	}
	if failIndex != -1 {
		return logList[failIndex:]
	}
	return nil
}
