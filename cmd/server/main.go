package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"

	"github.com/gorilla/mux"

	"github.com/DulanDev/GoImager/internal/config"
	"github.com/DulanDev/GoImager/internal/handler"
	"github.com/DulanDev/GoImager/internal/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load failed: %v\n", err)
		os.Exit(1)
	}

	log := middleware.NewLogger(cfg.Logging.Level, cfg.Logging.Format)
	log.Info("starting GoImager", "port", cfg.Server.Port, "version", handler.Version)
	warnMissingTools(&cfg, log)

	srv := handler.New(cfg, log)
	r := mux.NewRouter()

	rl := middleware.NewRateLimiter(cfg.RateLimit.RPS, cfg.RateLimit.RPM)

	r.Use(middleware.ProcessingTimeAdapter())
	r.Use(middleware.AuthAdapter(cfg.Auth.APIKey))
	r.Use(rl.Adapter())
	r.Use(middleware.RequestLogger(log))

	r.HandleFunc("/health", srv.Health).Methods("GET")
	r.HandleFunc("/info", srv.Info).Methods("GET", "POST")
	r.HandleFunc("/resize", srv.Resize).Methods("POST")
	r.HandleFunc("/convert", srv.Convert).Methods("POST")
	r.HandleFunc("/optimize", srv.Optimize).Methods("POST")
	r.HandleFunc("/thumbnail", srv.Thumbnail).Methods("POST")
	// /process is governed by Sign (HMAC-signed URLs), not the global
	// Bearer API_KEY middleware — see Authentication Model in specs.
	r.Handle("/process", middleware.VerifySign(cfg.Auth.SigningKeysList(), http.HandlerFunc(srv.Process))).Methods("GET")
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("GoImager"))
	}).Methods("GET")

	addr := ":" + cfg.Server.Port
	log.Info("listening", "addr", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func warnMissingTools(cfg *config.Config, log *slog.Logger) {
	if cfg.Optimizer.PngquantPath != "" && !toolExists(cfg.Optimizer.PngquantPath) {
		log.Warn("pngquant not found; PNG optimization falls back to re-encode", "path", cfg.Optimizer.PngquantPath)
	}
	if cfg.Optimizer.MozjpegPath != "" && !toolExists(cfg.Optimizer.MozjpegPath) {
		log.Warn("mozjpeg cjpeg not found; JPEG optimization falls back to Go encoder", "path", cfg.Optimizer.MozjpegPath)
	}
	if cfg.Optimizer.CwebpPath != "" && !toolExists(cfg.Optimizer.CwebpPath) {
		log.Warn("cwebp not found; WebP output unavailable", "path", cfg.Optimizer.CwebpPath)
	}
	if cfg.Optimizer.AvifPath != "" && !toolExists(cfg.Optimizer.AvifPath) {
		log.Warn("avifenc not found; AVIF output unavailable, falls back to JPEG", "path", cfg.Optimizer.AvifPath)
	}
}

func toolExists(path string) bool {
	_, err := exec.LookPath(path)
	return err == nil
}