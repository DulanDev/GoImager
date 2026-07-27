package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Auth enforces an API key supplied via the Authorization: Bearer <key>
// header on all requests. The path /health (and root "/") is always
// allowed so orchestrators can probe without credentials. When apiKey
// is empty, every request is allowed (auth disabled).
func Auth(apiKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if apiKey == "" || publicAuthPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		provided := extractBearer(r)
		if provided != apiKey {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("WWW-Authenticate", "Bearer")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "missing or invalid API key",
				"code":  "UNAUTHORIZED",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func publicAuthPath(p string) bool {
	return p == "/health" || p == "/"
}

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(h, prefix))
}

// AuthAdapter returns a gorilla/mux-compatible middleware function.
func AuthAdapter(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return Auth(apiKey, next)
	}
}