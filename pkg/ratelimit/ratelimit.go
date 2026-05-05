// now - lastRefull = duration
// tokens += duration * rate

package ratelimit

import (
	"sync"
	"time"
)

type RateLimiter struct {
	keys map[string]*bucket
	su   sync.Mutex
}

type bucket struct {
	lastRefill time.Time
	tokens     float64
	rate       float64 // tokens/time
	max        float64
	su         sync.Mutex
}

func (b *bucket) Allow() bool {
	b.su.Lock()
	defer b.su.Unlock()
	duration := time.Since(b.lastRefill)
	b.tokens += duration.Seconds() * b.rate
	b.lastRefill = time.Now()

	if b.tokens > b.max {
		b.tokens = b.max
	}

	if b.tokens > 1 {
		b.tokens -= 1
		return true
	} else {
		return false
	}

}
