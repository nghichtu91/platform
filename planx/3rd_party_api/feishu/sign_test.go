package feishu

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/nghichtu91/platform/share/planx/timeutil"
)

func originalGenSign(secret string, timestamp int64) (string, error) {
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

func TestGenSign(t *testing.T) {
	secret := "5XmM1vTD7YNUI3wg7Tdouc"

	ticker := timeutil.Timer10MS.After(300 * time.Millisecond)
	i := 0

	InitFeishu(secret)

loop:
	for {
		select {
		case <-ticker:
			sign, _ := originalGenSign(secret, time.Now().Unix())
			assert.Equal(t, sign, getSign())

			i++
			if i >= 64 {
				break loop
			}

			ticker = timeutil.Timer10MS.After(300 * time.Millisecond)
		}
	}

}
