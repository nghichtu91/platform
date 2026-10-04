package dynamodb

import (
	"github.com/cenk/backoff"
	"github.com/nghichtu91/platform/share/planx/awshelper"
)

func NewExponentialBackOffSleepFirst() *backoff.ExponentialBackOff {
	return awshelper.NewExponentialBackOffSleepFirst()
}
