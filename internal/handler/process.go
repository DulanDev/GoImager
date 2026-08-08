package handler

import (
	"net/http"
	"strconv"

	"github.com/DulanDev/GoImager/internal/service"
)

func (s *Server) Process(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	p := service.ProcessParams{
		Src:    q.Get("src"),
		Mode:   q.Get("mode"),
		Format: q.Get("format"),
		Flip:   q.Get("flip"),
	}
	p.W = queryInt(q, "w")
	p.H = queryInt(q, "h")
	p.Q = queryInt(q, "q")
	p.Rotate = queryInt(q, "rotate")
	p.Blur = queryFloat(q, "blur")
	p.Sharp = queryFloat(q, "sharp")

	out, ct, err := service.Process(p, s.Cfg, s.HTTPClient)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", ct)
	_, _ = w.Write(out)
}

func queryInt(q interface {
	Get(string) string
}, key string) int {
	v := q.Get(key)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}

func queryFloat(q interface {
	Get(string) string
}, key string) float64 {
	v := q.Get(key)
	if v == "" {
		return 0
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}
	return f
}
