package k8s

import (
	"errors"
)

var (
	ErrNotInitialized = errors.New("client not initialized")

	ErrInvalidReplicas     = errors.New("invalid number of replicas")
	ErrInvalidNamespace    = errors.New("invalid namespace")
	ErrInvalidResourceName = errors.New("invalid deployment")

	ErrWatchInitFailed = errors.New("watch init failed")
	ErrWatchFailed     = errors.New("watch failed")
	ErrWatchInterrupt  = errors.New("watch interrupted")
	ErrWatchInvalidRev = errors.New("invalid resource version")
)
