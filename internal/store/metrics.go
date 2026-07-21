package store

import "sync/atomic"

type Metrics struct {
	Hits        atomic.Int64
	Misses      atomic.Int64
	Evictions   atomic.Int64
	Expirations atomic.Int64
}
