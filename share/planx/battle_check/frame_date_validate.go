package battle_check

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"github.com/nghichtu91/platform/share/planx/tilogs"
)

type FramValidateData struct {
	frameID       int16
	result        []byte
	resultContext string
	accid         string
	name          string
}

func NewFrameValidateData(framID int16, result []byte, resultContext, accid, name string) FramValidateData {
	return FramValidateData{
		frameID:       framID,
		result:        result,
		resultContext: resultContext,
		accid:         accid,
		name:          name,
	}
}

func (v *FrameDataValidator) Upload(datas []FramValidateData) {
	for _, data := range datas {
		v.write(data)
	}
}

type ClientCompareData struct {
	name         string
	frameResults []FramValidateData
}

type OssConfig struct {
	OSS_Bucket    string
	OSS_EndPoint  string
	OSS_AccessKey string
	OSS_SecretKey string
}

func (v *FrameDataValidator) InitOssConfig(bucket, endPoint, accessKey, secretKey string) {
	v.ossConfig =
		OssConfig{
			OSS_Bucket:    bucket,
			OSS_EndPoint:  endPoint,
			OSS_AccessKey: accessKey,
			OSS_SecretKey: secretKey,
		}
}

type FrameDataValidator struct {
	gid                uint
	sid                uint
	isSynch            bool
	frameAllClientData map[string]*ClientCompareData
	recvFramData       chan FramValidateData
	quit               chan struct{}
	roomPlayerNun      byte
	roomID             uint32
	ossConfig          OssConfig
}

func GenFrameDataValidator(roomID uint32, sid, gid uint) *FrameDataValidator {
	return &FrameDataValidator{
		gid:                gid,
		sid:                sid,
		isSynch:            true,
		frameAllClientData: make(map[string]*ClientCompareData),
		recvFramData:       make(chan FramValidateData, 128),
		quit:               make(chan struct{}, 1),
		roomPlayerNun:      byte(2),
		roomID:             roomID,
	}
}

func (validator *FrameDataValidator) End() {
	close(validator.quit)
}

func (validator *FrameDataValidator) write(data FramValidateData) {
	validator.recvFramData <- data
}

func (validator *FrameDataValidator) addCompareData(data FramValidateData) {
	acid := data.accid
	name := data.name
	frameData, ok := validator.frameAllClientData[acid]
	if !ok {
		frameData = &ClientCompareData{}
		frameData.name = name
		validator.frameAllClientData[acid] = frameData
	}
	frameData.frameResults = append(frameData.frameResults, data)
}

func (validator *FrameDataValidator) Start() {
	go func() {
		defer tilogs.PanicCatcher("FrameDataValidator.recvFramData error...")
		for {
			select {
			case recvDate := <-validator.recvFramData:
				validator.addCompareData(recvDate)
			case <-validator.quit:
				validator.compareAndCheckFrameDate()
				return
			}
		}
	}()
}

func compare(c1, c2 *ClientCompareData) bool {
	var size int
	if len(c1.frameResults) >= len(c2.frameResults) {
		size = len(c2.frameResults)
	} else {
		size = len(c1.frameResults)
	}
	for i := 0; i < size; i++ {
		if bytes.Compare(c1.frameResults[i].result, c2.frameResults[i].result) != 0 {
			return false
		}
	}
	return true
}

func (v *FrameDataValidator) compareAndCheckFrameDate() {
	var tmp *ClientCompareData
	var index = 0
	for _, value := range v.frameAllClientData {
		if tmp == nil {
			tmp = value
		} else if !compare(tmp, value) {
			v.isSynch = false
			break
		}
		index++
	}
	if v.isSynch {
		tilogs.L().Infof("room %d frame compare same...", v.roomID)
		return
	}
	tilogs.L().Infof("room %d frame compare not same...", v.roomID)
	if index > 20 {
		index = index - 20
	} else {
		index = 0
	}

	tNow := time.Now()
	timeDayStr := tNow.Format("2006-01-02")
	timeStr := tNow.Format("2006-01-02 15:04:05")
	path := fmt.Sprintf("framecompare/%d/%s/%s/Ser_%d_GRoomId_%d/", v.gid, timeDayStr, timeStr, v.sid, v.roomID)
	// 创建OSSClient实例。
	client, err := oss.New(v.ossConfig.OSS_EndPoint, v.ossConfig.OSS_AccessKey, v.ossConfig.OSS_SecretKey)
	if err != nil {
		tilogs.L().Errorf(err.Error())
		return
	}

	// 获取存储空间。
	bucket, err := client.Bucket(v.ossConfig.OSS_Bucket)
	if err != nil {
		tilogs.L().Errorf(err.Error())
		return
	}
	for _, value := range v.frameAllClientData {
		writeFile(value.name, value, index, path, bucket)
	}

}

func writeFile(name string, data *ClientCompareData, index int, path string, bucket *oss.Bucket) {
	uploadKey := path + name + ".txt"
	var result []byte
	for ; index < len(data.frameResults); index++ {
		v := data.frameResults[index]
		result = append(result, v.result...)
	}
	err := bucket.PutObject(uploadKey, bytes.NewReader(result))
	if err != nil {
		tilogs.L().Errorf(err.Error())
		return
	}
}

type ValidateInfo struct {
	accName string
}

func (validator *FrameDataValidator) Compare(client1, client2 string) bool {
	isSame := strings.Compare(client1, client2)
	if isSame == 0 {
		return true
	}
	return false
}
