package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiterAllowsUnderLimit(t *testing.T) {
	rl := NewRateLimiter(2, 0)
	h := rl.Adapter()(okHandler())
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/resize", nil)
		req.RemoteAddr = "1.2.3.4:5678"
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d should pass, got %d", i, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resize", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("3rd request should be limited, got %d", rec.Code)
	}
}

func TestRateLimiterDisabledOnZero(t *testing.T) {
	rl := NewRateLimiter(0, 0)
	if rl.rate != 0 {
		t.Fatal("rate should be 0 when disabled")
	}
	for i := 0; i < 50; i++ {
		if !rl.Allow("9.9.9.9") {
			t.Fatalf("disabled limiter should allow request %d", i)
		}
	}
}

func TestRateLimiterHealthBypass(t *testing.T) {
	rl := NewRateLimiter(0, 1)
	h := rl.Adapter()(okHandler())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("health bypass failed, got %d", rec.Code)
	}
}

func TestRateLimiterRPM(t *testing.T) {
	rl := NewRateLimiter(0, 1)
	if rl.rate <= 0 {
		t.Fatal("rpm mode should set rate")
	}
}