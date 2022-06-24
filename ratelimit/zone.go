package ratelimit

import (
	"time"
	"fmt"

	sw "github.com/RussellLuo/slidingwindow"
	"github.com/hashicorp/golang-lru"
)

type Zone struct {
	limiters *lru.Cache

	rateSize  time.Duration
	rateLimit int64
}

func NewZone(size int, rateSize time.Duration, rateLimit int64) (*Zone, error) {
	cache, err := lru.New(size)
	if err != nil {
		return nil, err
	}
	return &Zone{
		limiters:  cache,
		rateSize:  rateSize,
		rateLimit: rateLimit,
	}, nil
}

// Purge is used to completely clear the zone.
func (z *Zone) Purge() {
	z.limiters.Purge()
}

func (z *Zone) Allow(key string) bool {
	lim, _, _ := z.getLimiter(key)
	return lim.Allow()
}

func (z *Zone) RateLimitPolicyHeader() string {
	return fmt.Sprintf("%d; w=%d", z.rateLimit, int(z.rateSize.Seconds()))
}

func (z *Zone) getLimiter(key string) (lim *sw.Limiter, ok, evict bool) {
	// If there is already a limiter for key, just return it.
	elem, ok := z.limiters.Peek(key)
	if ok {
		return elem.(*sw.Limiter), true, false
	}

	lim, _ = sw.NewLimiter(z.rateSize, z.rateLimit, func() (sw.Window, sw.StopFunc) {
		// NewLocalWindow returns an empty stop function, so it's
		// unnecessary to call it later.
		return sw.NewLocalWindow()
	})
	// Try to add lim as the limiter for key.
	ok, evict = z.limiters.ContainsOrAdd(key, lim)

	if ok {
		// The limiter for key has been added by someone else just now.
		// We should use the limiter rather than our lim.
		elem, _ = z.limiters.Peek(key)
		lim = elem.(*sw.Limiter)
	}

	return
}
