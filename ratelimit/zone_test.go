package ratelimit

import (
	"testing"
	"time"

	sw "github.com/RussellLuo/slidingwindow"
)

func TestZone_getLimiter(t *testing.T) {
	zone, _ := NewZone(2, time.Second, 10)

	cases := []struct {
		key   string
		ok    bool
		evict bool
	}{
		{"key1", false, false},
		{"key1", true, false},
		{"key2", false, false},
		{"key3", false, true},
	}

	for _, c := range cases {
		lim, ok, evict := zone.getLimiter(c.key)
		if lim == nil {
			t.Fatalf("Limiter is nil")
		}

		if ok != c.ok {
			t.Fatalf("Found: got (%#v), want (%#v)", ok, c.ok)
		}

		if evict != c.evict {
			t.Fatalf("Evict: got (%#v), want (%#v)", evict, c.evict)
		}
	}
}

func TestZone_getLimiterConcurrently(t *testing.T) {
	test := func(n int) {
		zone, _ := NewZone(1, time.Second, 10)
		key := "key1"

		limC := make(chan *sw.Limiter, n)
		startC := make(chan struct{})
		for i := 0; i < n; i++ {
			go func() {
				<-startC
				lim, _, _ := zone.getLimiter(key)
				limC <- lim
			}()
		}

		// Send a START signal to all the goroutines.
		close(startC)

		var gotLims []*sw.Limiter
		for i := 0; i < n; i++ {
			// Collect all the result limiters.
			gotLims = append(gotLims, <-limC)
		}

		elem, ok := zone.limiters.Peek(key)
		if !ok {
			t.Fatalf("Found no limiter")
		}
		wantLim := elem.(*sw.Limiter)

		for _, lim := range gotLims {
			if lim != wantLim {
				t.Fatalf("Limiter: got (%#v), want (%#v)", lim, wantLim)
			}
		}
	}

	cases := []struct {
		name   string
		degree int
	}{
		{"concurrency-degree-8", 8},
		{"concurrency-degree-32", 32},
		{"concurrency-degree-64", 64},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			test(c.degree)
		})
	}
}
