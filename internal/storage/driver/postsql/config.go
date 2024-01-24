package postsql

import "time"

const (
	defaultRetryTimeout  = time.Second * 1
	defaultRetryInterval = time.Millisecond * 1
)
