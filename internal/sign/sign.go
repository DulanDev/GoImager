// Package sign contains the canonicalization, signing and verification
// primitives for /process signed URLs. It is the single source of truth:
// internal/middleware uses it to verify incoming requests, cmd/gensig uses
// it to produce signed URLs for manual testing, and the algorithm documented
// in specs/product-specs.md → Authentication Model mirrors it exactly.
//
// Canonical form:
//
//	/process?<url.Values.Encode()>
//
// where url.Values.Encode() sorts keys lexicographically and percent-encodes
// using Go's standard application/x-www-form-urlencoded rules (spaces as
// '+'). The "sig" parameter, if present, is dropped before encoding.
package sign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
)

// Path is the only URL path this package knows how to sign/verify.
const Path = "/process"

// Canonical returns the string that is MAC'd for a /process URL: the path
// followed by "?" and the query string with `sig` removed and re-encoded in
// sorted form. Pass the request's full query (including sig, if present) —
// Canonical strips it.
func Canonical(q url.Values) string {
	clone := make(url.Values, len(q))
	for k, vs := range q {
		if k == "sig" {
			continue
		}
		clone[k] = vs
	}
	return Path + "?" + clone.Encode()
}

// Compute returns the lowercase hex HMAC-SHA256 of `canonical` under `key`.
func Compute(key, canonical string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}

// Valid reports whether `provided` (hex) matches the HMAC of `canonical`
// under any of `keys`. Constant-time compare per key. Returns false if no
// keys are non-empty.
func Valid(keys []string, canonical, provided string) bool {
	if provided == "" {
		return false
	}
	for _, k := range keys {
		if k == "" {
			continue
		}
		mac := hmac.New(sha256.New, []byte(k))
		mac.Write([]byte(canonical))
		computed := hex.EncodeToString(mac.Sum(nil))
		if hmac.Equal([]byte(computed), []byte(provided)) {
			return true
		}
	}
	return false
}
