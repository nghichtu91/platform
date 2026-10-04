package yidun

import (
	"github.com/nghichtu91/platform/share/planx/timeutil"
)

// Result 从易盾返回值中解析参数
type Result struct {
	// 0 通过，1 嫌疑，2 不通过
	Ret int

	// 不通过原因
	Reason string

	// 是否由于充值积分达到要求放行
	IsPassByScore bool

	// 封禁原因是否为其他，其他表示账号被封禁
	IsOther bool

	// 是否涉政，有额外处理逻辑
	IsPolitics bool

	// 是否发送飞书
	IsSendFeishu bool

	// 发往飞书的url
	FeishuUrl string
}

func HandleLabels(ret int, labels []*Lable, fnAssignFeishuUrl func(int64) string) (*Result, bool) {
	r := &Result{
		Ret:          ret,
		IsSendFeishu: ret == RetFail,
	}

	//是否是需要转为仅战区可见的跨服消息
	needCross2Zone := false
	for _, ele := range labels {
		if ele == nil {
			continue
		}

		switch ele.Lable {
		case 100: // 色情
			if !timeutil.IsAsiaShanghaiTZ() {
				r.IsSendFeishu = false
			}
		case 500: // 涉政
			if ele.Level == 1 || ele.Level == 2 { // 涉政嫌疑级别也拦下
				r.Ret = RetFail
				r.IsPolitics = true
			}
		case 900: // 被封禁
			r.IsOther = true
		}

		//嫌疑且暴恐，则认为是需要仅战区可见的跨服消息
		if ele.Level == 1 && ele.Lable == 300 {
			needCross2Zone = true
		}

		// 添加失败理由
		r.Reason += Labels[ele.Lable] + "(" + Results[ele.Level] + ")"

		// 分配对应的飞书地址
		r.FeishuUrl = fnAssignFeishuUrl(ele.Lable)
	}

	return r, needCross2Zone
}
