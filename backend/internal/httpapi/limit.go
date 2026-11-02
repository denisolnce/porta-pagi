package httpapi

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// A village-sized rate limiter: a map in memory, cleared by a restart. It
// exists so a code or a PIN cannot be guessed by a script from one address,
// not to survive a serious attacker; the wait per house (auth.go § PINs) is
// what holds against many.
type limiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func newLimiter() *limiter { return &limiter{hits: map[string][]time.Time{}} }

func (l *limiter) allow(key string, max int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	kept := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if now.Sub(t) < window {
			kept = append(kept, t)
		}
	}
	if len(l.hits) > 5000 { // never grow without bound
		l.hits = map[string][]time.Time{}
	}
	l.hits[key] = append(kept, now)
	return len(kept) < max
}

// clientIP prefers the address Caddy forwards, since the backend only ever
// sees the proxy otherwise.
func clientIP(r *http.Request) string {
	if f := r.Header.Get("X-Forwarded-For"); f != "" {
		return strings.TrimSpace(strings.Split(f, ",")[0])
	}
	if h, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return h
	}
	return r.RemoteAddr
}
