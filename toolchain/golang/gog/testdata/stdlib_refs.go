package testdata

import "sync/atomic"

func _() {
	atomic.AddInt32(nil, 123)
}
