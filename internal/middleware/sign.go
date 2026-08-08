package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/DulanDev/GoImager/internal/sign"
)

const (
	signatureParam = "sig"
	expiresParam   = "exp"
)

// VerifySign enforces HMAC-SHA256 signed URLs for /process.
//
// keys is the list of accepted signing keys; the first one that validates
// the request wins. When the list is empty (or contains only empty strings)
// signing is disabled and every request passes through — intended for
// private-network deployments where the network/reverse proxy is the auth.
//
// Per-request requirements when signing is enabled:
//   - ?exp=<unix seconds>      expiry, validated against the server clock
//   - ?sig=<hex hmac-sha256>   signature over sign.Canonical(query)
//
// Status codes:
//   - 410 Gone             — ?exp is in the past
//   - 401 Unauthorized     — missing/invalid exp, missing sig, bad signature
//
// See specs/product-specs.md → Authentication Model for the full rationale
// and reference signing snippets in Go and JavaScript.
func VerifySign(keys []string, next http.Handler) http.Handler {
	valid := make([]string, 0, len(keys))
	for _, k := range keys {
		if k != "" {
			valid = append(valid, k)
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(valid) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		q := r.URL.Query()
		sig := q.Get(signatureParam)
		exp := q.Get(expiresParam)
		if sig == "" || exp == "" {
			rejectSign(w, http.StatusUnauthorized, "UNAUTHORIZED")
			return
		}
		expUnix, err := strconv.ParseInt(exp, 10, 64)
		if err != nil {
			rejectSign(w, http.StatusUnauthorized, "UNAUTHORIZED")
			return
		}
		if time.Now().Unix() > expUnix {
			rejectSign(w, http.StatusGone, "EXPIRED")
			return
		}
		canonical := sign.Canonical(q)
		if !sign.Valid(valid, canonical, sig) {
			rejectSign(w, http.StatusUnauthorized, "UNAUTHORIZED")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func rejectSign(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "invalid or missing signature",
		"code":  code,
	})
}
