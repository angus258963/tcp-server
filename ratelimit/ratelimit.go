package ratelimit

import (
	"sync"
	"time"
)

type Ratelimit struct {
	bucket  int64
	count   int64
	limit   int64
	updated int64
	sync.Mutex
}

// New creates a ratelimit, it serves bucket per limit msecs
func New(bucket int64, limit int64) *Ratelimit {
	return &Ratelimit{
		bucket: bucket,
		limit:  limit,
	}
}

func (rate *Ratelimit) Acquire() bool {
	defer rate.Unlock()
	rate.Lock()
	nowMs := time.Now().UnixNano() / time.Millisecond.Nanoseconds()
	// according to buckets/limit, calculate the token ratio
	ratio := (nowMs - rate.updated) * 100 / rate.limit
	completedTokens := (ratio * rate.bucket) / 100
	// add tokens to bucket
	if completedTokens > 0 {
		rate.updated = nowMs
		rate.count += completedTokens
		// count can't greater than bucket
		if rate.count >= rate.bucket {
			rate.count = rate.bucket
		}
	}
	if rate.count > 0 {
		rate.count--
		return true
	}
	return false
}

func (rate *Ratelimit) Limit() int64 {
	return rate.limit
}
func (rate *Ratelimit) Bucket() int64 {
	return rate.bucket
}
