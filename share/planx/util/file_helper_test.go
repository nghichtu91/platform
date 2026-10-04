package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWalkWithAllFiles(t *testing.T) {
	fn, err := WalkWithAllFiles("..", false, true, true)
	assert.Nil(t, err)
	t.Logf("fn: %v", fn)

	fn, err = WalkWithAllFiles("..", true, false, false)
	assert.Nil(t, err)
	t.Logf("fn: %v", fn)
}
