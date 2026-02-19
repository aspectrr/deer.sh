package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimitByIP_AllowsBurst(t *testing.T) {
	handler := rateLimitByIP(1, 3)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	handler := rateLimitByIP(0.001, 2)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	handler := rateLimitByIP(0.001, 1)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
