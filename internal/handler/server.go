package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/DulanDev/GoImager/internal/config"
)

type Server struct {
	Cfg        config.Config
	Log        *slog.Logger
	HTTPClient *http.Client
}

func New(cfg config.Config, log *slog.Logger) *Server {
	return &Server{
		Cfg: cfg,
		Log: log,
		HTTPClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

type errResp struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errResp{Error: msg, Code: code})
}

func (s *Server) maxBytes() int64 {
	if s.Cfg.Server.MaxFileSizeMB <= 0 {
		return 20 << 20
	}
	return int64(s.Cfg.Server.MaxFileSizeMB) << 20
}

// parseMultipartForm wraps r.ParseMultipartForm with a clearer error
// contract: oversize bodies return PAYLOAD_TOO_LARGE; anything else
// (missing body, wrong Content-Type, bad boundary — e.g. a REST Client
// GET that strips the body) returns INVALID_MULTIPART with the underlying
// stdlib error so the caller can debug.
func (s *Server) parseMultipartForm(w http.ResponseWriter, r *http.Request) bool {
	if err := r.ParseMultipartForm(s.maxBytes()); err != nil {
		msg := err.Error()
		if msg == "http: POST body too large" || msg == "request body too large" {
			writeError(w, http.StatusBadRequest, "PAYLOAD_TOO_LARGE", "request body exceeds max file size")
			return false
		}
		s.Log.Warn("multipart parse failed",
			"err", msg,
			"maxBytes", s.maxBytes(),
			"content_type", r.Header.Get("Content-Type"),
			"method", r.Method)
		writeError(w, http.StatusBadRequest, "INVALID_MULTIPART",
			"expected multipart/form-data with an image field; got "+msg)
		return false
	}
	return true
}

func (s *Server) defaultQuality() int {
	if s.Cfg.Quality.Default <= 0 {
		return 85
	}
	return s.Cfg.Quality.Default
}

func (s *Server) optimizerCfg() config.Optimizer {
	return s.Cfg.Optimizer
}
