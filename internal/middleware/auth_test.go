package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthDisabledWhenNoKey(t *testing.T) {
	h := Auth("", okHandler())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resize", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (auth disabled)", rec.Code)
	}
}

func TestAuthHealthBypass(t *testing.T) {
	h := Auth("secret", okHandler())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("health should bypass auth, got %d", rec.Code)
	}
}

func TestAuthProcessBypass(t *testing.T) {
	h := Auth("secret", okHandler())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/process?src=x", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("/process should bypass Bearer auth (governed by Sign), got %d", rec.Code)
	}
}

func TestAuthDocsBypass(t *testing.T) {
	h := Auth("secret", okHandler())
	cases := []string{
		"/openapi.yaml",
		"/openapi.json",
		"/docs",
		"/docs/",
		"/docs/swagger-ui.css",
		"/docs/swagger-ui-bundle.js",
		"/docs/favicon.ico",
	}
	for _, p := range cases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, p, nil)
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("path %q should bypass Bearer auth, got %d", p, rec.Code)
		}
	}
}

func TestAuthRejectsMissing(t *testing.T) {
	h := Auth("secret", okHandler())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resize", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("missing key should 401, got %d", rec.Code)
	}
}

func TestAuthRejectsWrongKey(t *testing.T) {
	h := Auth("secret", okHandler())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resize", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("wrong key should 401, got %d", rec.Code)
	}
}

func TestAuthApprovesValidKey(t *testing.T) {
	h := Auth("secret", okHandler())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resize", nil)
	req.Header.Set("Authorization", "Bearer secret")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("valid key should 200, got %d", rec.Code)
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}