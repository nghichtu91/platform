package chat

import (
	"fmt"

	uuid "github.com/satori/go.uuid"
	"github.com/nghichtu91/platform/share/planx/secure"
)

func GenerateChatToken(timeStamp int64) (string, string) {
	//生成LoginToken
	uid := uuid.NewV4().String()
	token := fmt.Sprintf("%s,%d", uid, timeStamp)
	enc_token := secure.Encode64ForNet([]byte(uid))
	return token, enc_token
}
