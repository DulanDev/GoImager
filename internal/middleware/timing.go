package middleware

import (
	"net/http"
	"strconv"
	"time"
)

// ProcessingTime sets the X-Processing-Time response header (in
// milliseconds, float with microsecond precision) on every response.
// The header is stamped at the first WriteHeader/Write flush, which for
// image handlers coincides with the end of processing since the full
// body is written in one call.
func ProcessingTime(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		tw := &timingWriter{ResponseWriter: w, start: start}
		next.ServeHTTP(tw, r)
	})
}

type timingWriter struct {
	http.ResponseWriter
	start       time.Time
	headerStamp bool
}

func (t *timingWriter) stamp() {
	if t.headerStamp {
		return
	}
	t.headerStamp = true
	ms := float64(time.Since(t.start).Microseconds()) / 1000.0
	t.ResponseWriter.Header().Set("X-Processing-Time", strconv.FormatFloat(ms, 'f', 3, 64))
}

func (t *timingWriter) WriteHeader(code int) {
	t.stamp()
	t.ResponseWriter.WriteHeader(code)
}

func (t *timingWriter) Write(p []byte) (int, error) {
	t.stamp()
	return t.ResponseWriter.Write(p)
}

// ProcessingTimeAdapter returns a gorilla/mux-compatible adapter.
func ProcessingTimeAdapter() func(http.Handler) http.Handler {
	return ProcessingTime
}
