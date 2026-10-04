package main

import (
	"fmt"
	"github.com/nghichtu91/platform/share/x/gift/modules/giftcode/giftcode_handler"
	"github.com/nghichtu91/platform/share/x/gift/test/tls/shard_2_gift_https/giftcode_api"
	"github.com/nghichtu91/platform/share/x/gift/test/tls/shard_2_gift_https/logic"
)

func main() {
	if err := logic.InitClient(); err != nil {
		fmt.Println(fmt.Sprintf("初始化Https客户端异常： %v", err))
		return
	}

	giftReq := &giftcode_handler.ClaimGiftCodeRequest{
		GiftCode:     "QHYVOXKBAQQPEP",
		HasUsedBatch: []int{},
		ChannelID:    "1",
		ACID:         "12:1:877172405",
	}
	giftResp := &giftcode_handler.ClaimGiftCodeResponse{}
	if err := logic.PostGiftCodeServer(giftcode_api.ClaimAPI, giftReq, giftResp); err != nil {
		fmt.Println(fmt.Sprintf("PostGiftCodeServer 异常： %v", err))
		return
	}

	fmt.Println(fmt.Sprintf("结果： %v", giftResp))
}
