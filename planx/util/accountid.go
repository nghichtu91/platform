package util

import (
	"strconv"
	"strings"
)

func ParseShardIDFromAcid(acid string) int {
	preIdx := strings.Index(acid, ":")
	if preIdx == -1 {
		return 0
	}

	acid = acid[preIdx+1:]

	suffixIdx := strings.Index(acid, ":")
	if suffixIdx == -1 {
		return 0
	}

	acid = acid[:suffixIdx]

	sid, _ := strconv.Atoi(acid)

	return sid
}
