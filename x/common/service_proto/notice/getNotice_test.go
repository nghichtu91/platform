package notice

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/nghichtu91/platform/share/planx/tilogs/zaplog"
)

func TestGetNotice(t *testing.T) {
	zaplog.InitZapLog("", nil, "")
	assert.Nil(t, GetNotice("http://121.196.106.98:8801/", "16", "0.0.0"))
}
