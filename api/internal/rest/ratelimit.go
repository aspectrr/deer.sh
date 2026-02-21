package rest

import (
	"log/slog"
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

// parseCIDRs parses a slice of CIDR strings into net.IPNet values.
// Invalid entries are skipped with a warning log.
func parseCIDRs(cidrs []string, logger *slog.Logger) []*net.IPNet {
	if logger == nil {
		logger = slog.Default()
	}
	var nets []*net.IPNet
	for _, c := range cidrs {
		_, ipNet, err := net.ParseCIDR(c)
		if err != nil {
			// Try as bare IP by appending /32 or /128.
			ip := net.ParseIP(c)
			if ip == nil {
				logger.Warn("skipping invalid trusted proxy CIDR", "cidr", c, "error", err)
				continue
			}
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			ipNet = &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}
		}
		nets = append(nets, ipNet)
	}
	return nets
}

// clientIP extracts the real client IP from a request. Proxy headers
// (X-Real-IP, X-Forwarded-For) are only trusted when RemoteAddr falls
// within one of the trustedProxies CIDRs.
func clientIP(r *http.Request, trustedProxies []*net.IPNet) string {
	remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	if remoteIP == "" {
		remoteIP = r.RemoteAddr
	}

	if len(trustedProxies) > 0 {
		parsed := net.ParseIP(remoteIP)
		if parsed != nil {
			for _, cidr := range trustedProxies {
				if cidr.Contains(parsed) {
					if xri := r.Header.Get("X-Real-IP"); xri != "" {
						return xri
					}
					if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
						ip, _, _ := strings.Cut(xff, ",")
						return strings.TrimSpace(ip)
					}
					break
				}
			}
		}
	}

	return remoteIP
}

// rateLimitByIP returns middleware that rate-limits requests per client IP.
// Proxy headers are only trusted when the direct connection comes from a
// trustedProxies CIDR.
//
// NOTE: Rate limit state is in-memory and per-process. In a multi-instance
// deployment, each instance maintains its own counters, so effective limits
// are multiplied by the number of instances. This is acceptable for
// single-instance deployments. For multi-instance, consider a shared store.
func rateLimitByIP(rps float64, burst int, trustedProxies []*net.IPNet) func(http.Handler) http.Handler {
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
			ip := clientIP(r, trustedProxies)

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
