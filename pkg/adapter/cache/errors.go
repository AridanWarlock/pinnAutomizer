package cache

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrIndexOutOfRange    = errors.New("index out of range")
	ErrRemovingLastBucket = errors.New("removing last bucket")
)
