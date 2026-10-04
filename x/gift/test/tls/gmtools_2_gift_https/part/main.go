package main

import (
	"encoding/json"
	"fmt"

	"github.com/nghichtu91/platform/share/x/gift/model/define"
	"github.com/nghichtu91/platform/share/x/gift/modules/common/util"
	"github.com/nghichtu91/platform/share/x/gift/modules/giftcode/giftcode_handler"
	"github.com/nghichtu91/platform/share/x/gift/test/tls/gmtools_2_gift_https/giftcode_api"
	"github.com/nghichtu91/platform/share/x/gift/test/tls/shard_2_gift_https/logic"
)

var a, e = fmt.Println(1)

// 生成测试
func main() {
	if err := logic.InitClient(); err != nil {
		fmt.Println(fmt.Sprintf("初始化Https客户端异常： %v", err))
		return
	}

	genGiftCodeInfo := define.GiftCodeInfo{
		BatchID:    10,
		GroupID:    1,
		Gid:        []string{"12", "13"},
		ChannelIds: []int{-1},
		GiftName:   "测试一",
		StartTime:  1604644786, // 2020-11-06 14:39:46
		EndTime:    1505853958, // 2020-11-08 14:39:46
		GiftType:   1,
		Items:      []define.Item{{Id: "1031", Count: 10}},
		GenCount:   100,
		TotalCount: 0,
		GenTime:    0,
	}

	contentJson, _ := json.Marshal(genGiftCodeInfo)

	giftReq := &giftcode_handler.GenGiftCodeRequest{
		Info: string(contentJson),
	}
	giftResp := &giftcode_handler.GenGiftCodeResponse{}
	if err := logic.PostGiftCodeServer(giftcode_api.GenGiftCode, giftReq, giftResp); err != nil {
		fmt.Println(fmt.Sprintf("PostGiftCodeServer 异常： %v", err))
		return
	}

	fmt.Println(fmt.Sprintf("结果： %v", giftResp))
}

// 追加测试。
func main2() {
	if err := logic.InitClient(); err != nil {
		fmt.Println(fmt.Sprintf("初始化Https客户端异常： %v", err))
		return
	}

	giftReq := &giftcode_handler.AddGiftCodeRequest{
		BatchID:     1,
		GroupID:     1,
		AppendCount: 100,
	}
	giftResp := &giftcode_handler.AddGiftCodeResponse{}
	if err := logic.PostGiftCodeServer(giftcode_api.AddGiftCode, giftReq, giftResp); err != nil {
		fmt.Println(fmt.Sprintf("PostGiftCodeServer 异常： %v", err))
		return
	}

	fmt.Println(fmt.Sprintf("结果： %v", giftResp))
}

// 下载测试
func main3() {
	if err := logic.InitClient(); err != nil {
		fmt.Println(fmt.Sprintf("初始化Https客户端异常： %v", err))
		return
	}

	giftReq := &giftcode_handler.DownloadGiftCodeRequest{
		BatchID: 1,
		GroupID: 1,
	}
	giftResp := &giftcode_handler.DownloadGiftCodeResponse{}
	if err := logic.PostGiftCodeServer(giftcode_api.DownloadGiftCode, giftReq, giftResp); err != nil {
		fmt.Println(fmt.Sprintf("PostGiftCodeServer 异常： %v", err))
		return
	}

	fmt.Println(fmt.Sprintf("结果： %v", giftResp))
}

// 历史测试
func main4() {
	if err := logic.InitClient(); err != nil {
		fmt.Println(fmt.Sprintf("初始化Https客户端异常： %v", err))
		return
	}

	giftReq := &giftcode_handler.QueryGenInfoRequest{
		Gid: []string{"13"},
	}
	giftResp := &giftcode_handler.QueryGenInfoResponse{}
	if err := logic.PostGiftCodeServer(giftcode_api.QueryGenGiftCode, giftReq, giftResp); err != nil {
		fmt.Println(fmt.Sprintf("PostGiftCodeServer 异常： %v", err))
		return
	}

	fmt.Println(fmt.Sprintf("结果： %v", giftResp))
}

// 查询测试。
func main5() {
	if err := logic.InitClient(); err != nil {
		fmt.Println(fmt.Sprintf("初始化Https客户端异常： %v", err))
		return
	}

	giftReq := &giftcode_handler.QueryGiftCodeRequest{
		BatchID: 1,
		GroupID: 2,
	}
	giftResp := &giftcode_handler.QueryGiftCodeResponse{}
	if err := logic.PostGiftCodeServer(giftcode_api.QueryGift, giftReq, giftResp); err != nil {
		fmt.Println(fmt.Sprintf("PostGiftCodeServer 异常： %v", err))
		return
	}

	fmt.Println(fmt.Sprintf("结果： %v", giftResp))
}

// 更新测试
func main6() {
	if err := logic.InitClient(); err != nil {
		fmt.Println(fmt.Sprintf("初始化Https客户端异常： %v", err))
		return
	}

	genGiftCodeInfo := define.GiftCodeInfo{
		BatchID:    1,
		GroupID:    3,
		Gid:        []string{"15", "14"},
		ChannelIds: []int{1, 2, 3, 4, 5},
		GiftName:   "测试一",
		StartTime:  1604644786, // 2020-11-06 14:39:46
		EndTime:    1604817586, // 2020-11-08 14:39:46
		GiftType:   1,
		Items:      []define.Item{},
		GenCount:   100,
		TotalCount: 0,
		GenTime:    0,
		UseCount:   1,
	}

	contentJson, _ := json.Marshal(genGiftCodeInfo)

	giftReq := &giftcode_handler.UpdateGiftCodeRequest{
		Info: string(contentJson),
	}
	giftResp := &giftcode_handler.UpdateGiftCodeResponse{}
	if err := logic.PostGiftCodeServer(giftcode_api.UpdateGift, giftReq, giftResp); err != nil {
		fmt.Println(fmt.Sprintf("PostGiftCodeServer 异常： %v", err))
		return
	}

	fmt.Println(fmt.Sprintf("结果： %v", giftResp))
}

// 销毁测试
func main7() {
	if err := logic.InitClient(); err != nil {
		fmt.Println(fmt.Sprintf("初始化Https客户端异常： %v", err))
		return
	}

	codes := []string{"Aloha233"}
	data := util.Codes2bytes(codes)

	giftReq := &giftcode_handler.DestroyGiftCodeRequest{
		BatchID: 1,
		GroupID: 5,
		Data:    data,
		Num:     1,
	}
	giftResp := &giftcode_handler.DestroyGiftCodeResponse{}
	if err := logic.PostGiftCodeServer(giftcode_api.DestroyGiftCode, giftReq, giftResp); err != nil {
		fmt.Println(fmt.Sprintf("PostGiftCodeServer 异常： %v", err))
		return
	}

	fmt.Println(fmt.Sprintf("结果： %v", giftResp))
}

func main8() {
	if err := logic.InitClient(); err != nil {
		fmt.Println(fmt.Sprintf("初始化Https客户端异常： %v", err))
		return
	}

	giftReq := &giftcode_handler.UpdateLARequest{
		Gid:    "16",
		OpType: giftcode_handler.Add,
	}
	giftResp := &giftcode_handler.UpdateLAResponse{}
	if err := logic.PostGiftCodeServer(giftcode_api.UpdateLA, giftReq, giftResp); err != nil {
		fmt.Println(fmt.Sprintf("PostGiftCodeServer 异常： %v", err))
		return
	}

	fmt.Println(fmt.Sprintf("结果： %v", giftResp))
}

func main9() {
	if err := logic.InitClient(); err != nil {
		fmt.Println(fmt.Sprintf("初始化Https客户端异常： %v", err))
		return
	}

	giftReq := &giftcode_handler.QueryLARequest{}
	giftResp := &giftcode_handler.QueryLAResponse{}
	if err := logic.PostGiftCodeServer(giftcode_api.QueryLA, giftReq, giftResp); err != nil {
		fmt.Println(fmt.Sprintf("PostGiftCodeServer 异常： %v", err))
		return
	}

	fmt.Println(fmt.Sprintf("结果： %v", giftResp))
}
