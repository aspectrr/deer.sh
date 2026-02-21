package rest

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimitByIP_AllowsBurst(t *testing.T) {
	handler := rateLimitByIP(1, 3, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d: got status %d, want %d", i, rr.Code, http.StatusOK)
		}
	}
}

func TestRateLimitByIP_RejectsAfterBurst(t *testing.T) {
	handler := rateLimitByIP(0.001, 2, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust burst
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/test", nil)
		req.RemoteAddr = "10.0.0.1:9999"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("burst request %d: got status %d, want %d", i, rr.Code, http.StatusOK)
		}
	}

	// Next request should be rejected
	req := httptest.NewRequest("POST", "/test", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("over-limit request: got status %d, want %d", rr.Code, http.StatusTooManyRequests)
	}
}

func TestRateLimitByIP_DifferentIPsIndependent(t *testing.T) {
	handler := rateLimitByIP(0.001, 1, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust IP A
	req := httptest.NewRequest("POST", "/test", nil)
	req.RemoteAddr = "1.1.1.1:1000"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("IP A first: got %d, want %d", rr.Code, http.StatusOK)
	}

	// IP A is now rate-limited
	req = httptest.NewRequest("POST", "/test", nil)
	req.RemoteAddr = "1.1.1.1:1000"
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("IP A second: got %d, want %d", rr.Code, http.StatusTooManyRequests)
	}

	// IP B should still be allowed
	req = httptest.NewRequest("POST", "/test", nil)
	req.RemoteAddr = "2.2.2.2:2000"
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("IP B first: got %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestRateLimitByIP_SpoofedHeaderIgnoredWithoutTrustedProxy(t *testing.T) {
	handler := rateLimitByIP(0.001, 1, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request with spoofed X-Forwarded-For - should use RemoteAddr
	req := httptest.NewRequest("POST", "/test", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("first request: got %d, want %d", rr.Code, http.StatusOK)
	}

	// Second request from same RemoteAddr but different spoofed header -
	// should still be rate-limited because we use RemoteAddr, not the header
	req = httptest.NewRequest("POST", "/test", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	req.Header.Set("X-Forwarded-For", "5.6.7.8")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("spoofed header should not bypass rate limit: got %d, want %d", rr.Code, http.StatusTooManyRequests)
	}
}

func TestRateLimitByIP_HeaderHonoredFromTrustedProxy(t *testing.T) {
	_, proxyNet, _ := net.ParseCIDR("10.0.0.0/8")
	trusted := []*net.IPNet{proxyNet}

	handler := rateLimitByIP(0.001, 1, trusted)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request from trusted proxy with X-Forwarded-For
	req := httptest.NewRequest("POST", "/test", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	req.Header.Set("X-Forwarded-For", "203.0.113.50")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("first request: got %d, want %d", rr.Code, http.StatusOK)
	}

	// Second request from same proxy but different real client - should be allowed
	req = httptest.NewRequest("POST", "/test", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	req.Header.Set("X-Forwarded-For", "203.0.113.51")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("different client via proxy: got %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestRateLimitByIP_XRealIPHonoredFromTrustedProxy(t *testing.T) {
	_, proxyNet, _ := net.ParseCIDR("10.0.0.0/8")
	trusted := []*net.IPNet{proxyNet}

	handler := rateLimitByIP(0.001, 1, trusted)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Request from trusted proxy with X-Real-IP (takes priority over XFF)
	req := httptest.NewRequest("POST", "/test", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	req.Header.Set("X-Real-IP", "203.0.113.50")
	req.Header.Set("X-Forwarded-For", "203.0.113.99")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("first request: got %d, want %d", rr.Code, http.StatusOK)
	}

	// Same X-Real-IP should be rate-limited
	req = httptest.NewRequest("POST", "/test", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	req.Header.Set("X-Real-IP", "203.0.113.50")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("same client via X-Real-IP: got %d, want %d", rr.Code, http.StatusTooManyRequests)
	}
}

func TestParseCIDRs(t *testing.T) {
	nets := parseCIDRs([]string{"10.0.0.0/8", "192.168.1.1", "invalid", "::1"}, nil)
	if len(nets) != 3 {
		t.Fatalf("expected 3 valid CIDRs, got %d", len(nets))
	}
}
