package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
)

const (
	maxDimension  = 10000
	defaultQuality = 85
	maxFileSizeMB = 20
)

// supportedFormats maps a request format name to its HTTP content type.
var supportedFormats = map[string]string{
	"jpeg": "image/jpeg",
	"png":  "image/png",
	"gif":  "image/gif",
}

// writeError emits a structured JSON error response per the API spec.
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": message,
		"code":  code,
	})
}

// parseDimension converts a form value to a non-negative int. It treats an
// empty string as 0 (auto-scale) and rejects anything non-numeric or negative.
func parseDimension(value, field string) (int, bool, string) {
	if value == "" {
		return 0, true, ""
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, false, field + " must be an integer"
	}
	if n < 0 {
		return 0, false, field + " must be non-negative"
	}
	if n > maxDimension {
		return 0, false, field + " exceeds max dimension of " + strconv.Itoa(maxDimension)
	}
	return n, true, ""
}

// normalizeFormat validates and lowercases a requested output format. An empty
// requested format returns ok=false with an empty contentType so callers can
// apply their own default.
func normalizeFormat(format string) (contentType string, ok bool) {
	ct, found := supportedFormats[format]
	if !found {
		return "", false
	}
	return ct, true
}