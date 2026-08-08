package main

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"github.com/DulanDev/GoImager/api"
	"github.com/DulanDev/GoImager/internal/config"
	"github.com/DulanDev/GoImager/internal/handler"
	"github.com/DulanDev/GoImager/internal/middleware"
)

func main() {
	os.Exit(run())
}

// run loads config, wires routes, and serves until SIGINT/SIGTERM or a
// fatal listen error. Returns the process exit code.
func run() int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load failed: %v\n", err)
		return 1
	}

	log := middleware.NewLogger(cfg.Logging.Level, cfg.Logging.Format)
	log.Info("starting GoImager", "port", cfg.Server.Port, "version", handler.Version)
	warnMissingTools(&cfg, log)
	warnOpenProcess(&cfg, log)

	srv := handler.New(cfg, log, api.OpenAPIYAML, sub(api.UIFS, "ui"))
	r := mux.NewRouter()

	rl := middleware.NewRateLimiter(cfg.RateLimit.RPS, cfg.RateLimit.RPM)

	r.Use(middleware.ProcessingTimeAdapter())
	r.Use(middleware.AuthAdapter(cfg.Auth.APIKey))
	r.Use(rl.Adapter())
	r.Use(middleware.RequestLogger(log))

	r.HandleFunc("/health", srv.Health).Methods("GET")
	r.HandleFunc("/info", srv.Info).Methods("POST")
	r.HandleFunc("/resize", srv.Resize).Methods("POST")
	r.HandleFunc("/convert", srv.Convert).Methods("POST")
	r.HandleFunc("/optimize", srv.Optimize).Methods("POST")
	r.HandleFunc("/thumbnail", srv.Thumbnail).Methods("POST")
	// /process is governed by Sign (HMAC-signed URLs), not the global
	// Bearer API_KEY middleware — see Authentication Model in specs.
	r.Handle("/process", middleware.VerifySign(cfg.Auth.SigningKeysList(), http.HandlerFunc(srv.Process))).Methods("GET")
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("GoImager"))
	}).Methods("GET")
	r.HandleFunc("/openapi.yaml", srv.OpenAPI).Methods("GET")
	r.HandleFunc("/openapi.json", srv.OpenAPI).Methods("GET")
	r.PathPrefix("/docs").Handler(srv.SwaggerUI()).Methods("GET")

	addr := ":" + cfg.Server.Port
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- httpSrv.ListenAndServe() }()

	log.Info("listening", "addr", addr)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Error("server stopped unexpectedly", "err", err)
			return 1
		}
	case sig := <-sigCh:
		log.Info("shutdown signal received", "signal", sig.String())
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			log.Error("graceful shutdown failed", "err", err)
			return 1
		}
		log.Info("server stopped cleanly")
	}
	return 0
}

// sub returns fsys rooted at dir, or nil if dir is absent. Used so we
// never serve the embed.FS root (which exposes both openapi.yaml and ui/).
func sub(fsys fs.FS, dir string) fs.FS {
	out, err := fs.Sub(fsys, dir)
	if err != nil {
		return nil
	}
	return out
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

// warnOpenProcess logs a loud warning when /process (browser-direct GET
// that fetches a remote src URL) is reachable without a signing key and
// the domain allowlist is the wildcard "*". That combination is an open
// SSRF oracle; the operator should enable signing and/or tighten
// ALLOWED_DOMAINS before exposing the service. SSRF on the resolved IP is
// still blocked at the dial layer (see internal/service/ssrf.go), but
// auth/allowlist policy is the first line of defence.
func warnOpenProcess(cfg *config.Config, log *slog.Logger) {
	wildcard := false
	for _, a := range cfg.Allowed {
		if a == "*" {
			wildcard = true
			break
		}
	}
	if !wildcard {
		return
	}
	if len(cfg.Auth.SigningKeysList()) > 0 {
		return
	}
	log.Warn("SECURITY: /process is open — wildcard ALLOWED_DOMAINS and no SIGNING_KEY set",
		"advice", "set SIGNING_KEY / SIGNING_KEYS and/or restrict ALLOWED_DOMAINS before exposing publicly")
}

func toolExists(path string) bool {
	_, err := exec.LookPath(path)
	return err == nil
}