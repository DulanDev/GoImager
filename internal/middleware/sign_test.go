package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DulanDev/GoImager/internal/sign"
)

// signURL builds a /process URL signed exactly as the verifier expects,
// using the shared internal/sign package for canonicalization + HMAC.
func signURL(t *testing.T, key string, q url.Values) string {
	t.Helper()
	q.Set("exp", strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10))
	q.Del("sig")
	q.Set("sig", sign.Compute(key, sign.Canonical(q)))
	return sign.Path + "?" + q.Encode()
}

func TestSignDisabledWhenNoKeys(t *testing.T) {
	h := VerifySign(nil, okHandler())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/process?src=x", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("empty keys = open; got %d", rec.Code)
	}
}

func TestSignDisabledWhenOnlyEmptyKeys(t *testing.T) {
	h := VerifySign([]string{"", ""}, okHandler())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/process?src=x", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("only-empty keys = open; got %d", rec.Code)
	}
}

func TestSignRejectsMissingSigAndExp(t *testing.T) {
	h := VerifySign([]string{"k"}, okHandler())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/process?src=x", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("missing sig/exp should 401, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"UNAUTHORIZED"`) {
		t.Errorf("body = %s", rec.Body.String())
	}
}

func TestSignRejectsBadExp(t *testing.T) {
	h := VerifySign([]string{"k"}, okHandler())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/process?src=x&exp=notanumber&sig=abc", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("bad exp should 401, got %d", rec.Code)
	}
}

func TestSignRejectsExpired(t *testing.T) {
	h := VerifySign([]string{"k"}, okHandler())
	rec := httptest.NewRecorder()
	q := url.Values{"src": {"x"}, "exp": {strconv.FormatInt(time.Now().Add(-time.Hour).Unix(), 10)}}
	q.Set("sig", sign.Compute("k", sign.Canonical(q)))
	req := httptest.NewRequest(http.MethodGet, sign.Path+"?"+q.Encode(), nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusGone {
		t.Errorf("expired should 410, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"EXPIRED"`) {
		t.Errorf("body = %s", rec.Body.String())
	}
}

func TestSignRejectsBadSignature(t *testing.T) {
	h := VerifySign([]string{"k"}, okHandler())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet,
		"/process?src=x&exp="+strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)+"&sig=deadbeef", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("bad sig should 401, got %d", rec.Code)
	}
}

func TestSignApprovesValidSignature(t *testing.T) {
	key := "supersecret"
	h := VerifySign([]string{key}, okHandler())
	urlStr := signURL(t, key, url.Values{"src": {"https://example.com/p.jpg"}, "w": {"800"}, "format": {"webp"}, "q": {"80"}})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, urlStr, nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("valid sig should 200, got %d; body=%s", rec.Code, rec.Body.String())
	}
}

func TestSignApprovesAnyOfMultipleKeys(t *testing.T) {
	keys := []string{"old", "new"}
	for _, k := range keys {
		h := VerifySign(keys, okHandler())
		urlStr := signURL(t, k, url.Values{"src": {"x"}})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, urlStr, nil)
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("key %q should validate, got %d", k, rec.Code)
		}
	}
}

func TestSignRejectsTamperedParam(t *testing.T) {
	key := "k"
	// Sign an honest URL, then mutate one param value; the original sig must
	// no longer match the recomputed canonical query.
	signed := signURL(t, key, url.Values{"src": {"x"}, "w": {"800"}})
	parsed, err := url.Parse(signed)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	q := parsed.Query()
	q.Set("w", "801") // change a param, keep stale sig/exp
	tampered := "/process?" + q.Encode()

	h := VerifySign([]string{key}, okHandler())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, tampered, nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("tampered param should 401, got %d", rec.Code)
	}
}
