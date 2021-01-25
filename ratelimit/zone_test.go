package ratelimit

import (
	"testing"
	"time"
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
