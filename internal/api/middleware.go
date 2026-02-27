package api

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultRequestBodyLimitBytes caps request payload size to prevent memory exhaustion.
	DefaultRequestBodyLimitBytes int64 = 1 << 20 // 1 MiB

	// DefaultRateLimitRequests is the default request budget per client IP and window.
	DefaultRateLimitRequests = 60

	// DefaultRateLimitWindow is the default throttle window.
	DefaultRateLimitWindow = time.Minute
)

const (
	securityHeaderNoSniff = "nosniff"
	securityHeaderNoFrame = "DENY"
	securityHeaderHSTS    = "max-age=63072000; includeSubDomains"
	securityHeaderCSP     = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'"
)

type ipWindow struct {
	start time.Time
	count int
}

type ipRateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	now     func() time.Time
	windows map[string]ipWindow
}

func newIPRateLimiter(limit int, window time.Duration, now func() time.Time) *ipRateLimiter {
	if limit <= 0 {
		limit = DefaultRateLimitRequests
	}
	if window <= 0 {
		window = DefaultRateLimitWindow
	}
	if now == nil {
		now = time.Now
	}

	return &ipRateLimiter{
		limit:   limit,
		window:  window,
		now:     now,
		windows: make(map[string]ipWindow),
	}
}

func (l *ipRateLimiter) allow(clientIP string) bool {
	now := l.now()
	if clientIP == "" {
		clientIP = "unknown"
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Drop stale IP windows to keep memory bounded.
	for ip, window := range l.windows {
		if now.Sub(window.start) >= 2*l.window {
			delete(l.windows, ip)
		}
	}

	window := l.windows[clientIP]
	if window.start.IsZero() || now.Sub(window.start) >= l.window {
		l.windows[clientIP] = ipWindow{
			start: now,
			count: 1,
		}
		return true
	}

	if window.count >= l.limit {
		return false
	}

	window.count++
	l.windows[clientIP] = window
	return true
}

// SecurityHeaders ensures API responses include baseline browser hardening headers.
func SecurityHeaders(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", securityHeaderNoSniff)
		w.Header().Set("X-Frame-Options", securityHeaderNoFrame)
		w.Header().Set("Strict-Transport-Security", securityHeaderHSTS)
		w.Header().Set("Content-Security-Policy", securityHeaderCSP)
		next.ServeHTTP(w, r)
	})
}

// BodySizeLimit caps request body size before handler processing.
func BodySizeLimit(limitBytes int64) func(http.Handler) http.Handler {
	if limitBytes <= 0 {
		limitBytes = DefaultRequestBodyLimitBytes
	}

	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.NotFoundHandler()
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, limitBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitPerIP throttles requests by client IP.
func RateLimitPerIP(limit int, window time.Duration) func(http.Handler) http.Handler {
	return rateLimitPerIPWithClock(limit, window, time.Now)
}

func rateLimitPerIPWithClock(limit int, window time.Duration, now func() time.Time) func(http.Handler) http.Handler {
	limiter := newIPRateLimiter(limit, window, now)

	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.NotFoundHandler()
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter.allow(clientIPFromRequest(r)) {
				next.ServeHTTP(w, r)
				return
			}

			retryAfterSeconds := int(limiter.window.Seconds())
			if retryAfterSeconds <= 0 {
				retryAfterSeconds = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		})
	}
}

func clientIPFromRequest(r *http.Request) string {
	forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		if len(parts) > 0 {
			clientIP := strings.TrimSpace(parts[0])
			if clientIP != "" {
				return clientIP
			}
		}
	}

	remoteAddr := strings.TrimSpace(r.RemoteAddr)
	if remoteAddr == "" {
		return "unknown"
	}

	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil && host != "" {
		return host
	}

	return remoteAddr
}
