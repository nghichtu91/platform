package etcd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetKeyLastName(t *testing.T) {
	tests := []struct {
		in  string
		out string
	}{
		{"", ""},
		{"/", ""},
		{"server_root/gid/switch/devdebug/crossx", "crossx"},
		{"server_root/gid/switch/devdebug/", "devdebug"},
	}

	for i, test := range tests {
		assert.Equal(t, test.out, GetKeyLastName(test.in), i)
	}
}
