package auth

import (
	"sync"
	"time"
)

// limiter is a small in-memory sliding-window rate limiter. State is lost on restart and
// not shared between instances, which is fine for a single-VPS deployment.
type limiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
	last time.Time // last prune
}

func newLimiter() *limiter { return &limiter{hits: map[string][]time.Time{}} }

// allow records a hit for key and reports whether at most limit hits happened within window.
func (l *limiter) allow(key string, limit int, window time.Duration, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Sub(l.last) > time.Hour {
		for k, ts := range l.hits {
			if len(ts) == 0 || now.Sub(ts[len(ts)-1]) > 24*time.Hour {
				delete(l.hits, k)
			}
		}
		l.last = now
	}
	cutoff := now.Add(-window)
	kept := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= limit {
		l.hits[key] = kept
		return false
	}
	l.hits[key] = append(kept, now)
	return true
}
