package cometx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTCPFakeHandShake(t *testing.T) {
	assert.Nil(t, Check("127.0.0.1:10001"))
}
