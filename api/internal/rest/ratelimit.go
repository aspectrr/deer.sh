package rest

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// rateLimitByIP returns middleware that rate-limits requests per client IP.
func rateLimitByIP(rps float64, burst int) func(http.Handler) http.Handler {
	var mu sync.Mutex
	limiters := make(map[string]*ipLimiter)

	// Periodically clean up stale entries. This goroutine is intentionally
	// process-scoped: rateLimitByIP is called at startup and lives for the
	// lifetime of the server, so no shutdown mechanism is needed.
	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, l := range limiters {
				if time.Since(l.lastSeen) > 10*time.Minute {
					delete(limiters, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.Header.Get("X-Real-IP")
			if ip == "" {
				if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
					ip, _, _ = strings.Cut(xff, ",")
					ip = strings.TrimSpace(ip)
				}
			}
			if ip == "" {
				ip, _, _ = net.SplitHostPort(r.RemoteAddr)
				if ip == "" {
					ip = r.RemoteAddr
				}
			}

			mu.Lock()
			l, ok := limiters[ip]
			if !ok {
				l = &ipLimiter{limiter: rate.NewLimiter(rate.Limit(rps), burst)}
				limiters[ip] = l
			}
			l.lastSeen = time.Now()
			mu.Unlock()

			if !l.limiter.Allow() {
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
