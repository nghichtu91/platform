package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/nghichtu91/platform/share/planx/cloud_db"
)

func Codes2bytes(codes []string) []byte {
	var resultBytes []byte
	buffer := bytes.NewBuffer(resultBytes)
	for _, code := range codes {
		buffer.WriteString(code)
		buffer.WriteString("\r\n")
	}
	return buffer.Bytes()
}

func Bytes2Codes(content []byte) []string {
	fmt.Println(strings.Split(string(content), "\r\n"))
	return nil
}

func main() {
	fmt.Println(Codes2bytes([]string{"QWE", "ASD"}))
	Bytes2Codes(Codes2bytes([]string{"QWE", "ASD"}))

	//// 初始化OSS数据库
	//db := InitOSSDB()

	//codes := []string{"QWER", "ASD", "ZX"}
	//resultJson, _ := json.Marshal(codes)

	//db.PutWithBucket("giftserver_codes/table1", resultJson, "jws2-battle-data")

	//names, err := db.ListObjectWithBucket("jws2-battle-data", "giftserver_codes/table1", 2, "giftserver_code/table")
	//if err != nil {
	//	fmt.Println(fmt.Sprintf("获取文件列表异常：%v", err))
	//}
	//fmt.Println(names)

	//dcodes := make([]string, 0, 3)
	//downLoad, _ := db.Get("giftserver_codes/table2")
	//json.Unmarshal(downLoad, &dcodes)
	//fmt.Println(dcodes)
	//
	//db.Close()
}

func InitOSSDB() cloud_db.CloudDb {
	giftCodeDB := cloud_db.NewCloudDB(cloud_db.CloudDbConfig{
		DbDriver:    cloud_db.CloudDb_Aliyun,
		Region:      "http://oss-cn-hangzhou.aliyuncs.com",
		Bucket:      "jws2-battle-data",
		CloudDbRoot: "12local",
		AccessKey:   "",
		SecretKey:   "",
	})
	if giftCodeDB == nil {
		return nil
	}
	if err := giftCodeDB.Open(); err != nil {
		fmt.Println(fmt.Sprintf("打开链接异常：%v", err))
		return nil
	}
	return giftCodeDB
}

func InitOBSDB() cloud_db.CloudDb {
	giftCodeDB := cloud_db.NewCloudDB(cloud_db.CloudDbConfig{
		DbDriver:    cloud_db.CloudDb_Huawei,
		Region:      "http://obs.cn-east-3.myhuaweicloud.com",
		Bucket:      "jws2-12-prod-data",
		CloudDbRoot: "12local",
		AccessKey:   "",
		SecretKey:   "",
	})
	if giftCodeDB == nil {
		return nil
	}
	if err := giftCodeDB.Open(); err != nil {
		fmt.Println(fmt.Sprintf("打开链接异常：%v", err))
		return nil
	}
	return giftCodeDB
}
