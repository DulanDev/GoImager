package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProcessingTimeHeaderSet(t *testing.T) {
	h := ProcessingTime(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resize", nil)
	h.ServeHTTP(rec, req)
	if rec.Header().Get("X-Processing-Time") == "" {
		t.Fatal("X-Processing-Time header missing")
	}
}

func TestProcessingTimeOnImplicitWrite(t *testing.T) {
	h := ProcessingTime(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("data"))
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resize", nil)
	h.ServeHTTP(rec, req)
	if rec.Header().Get("X-Processing-Time") == "" {
		t.Fatal("header missing on implicit write")
	}
	if rec.Body.String() != "data" {
		t.Errorf("body mismatch: %q", rec.Body.String())
	}
}