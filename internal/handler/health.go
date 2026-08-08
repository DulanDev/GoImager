package handler

import (
	"encoding/json"
	"net/http"
)

// Version is the service version. Overridden at build time via ldflags:
//
//	go build -ldflags "-X github.com/DulanDev/GoImager/internal/handler.Version=$(git describe --tags --always --dirty)" ./cmd/server
//
// Defaults to "dev" for `go run` and bare `docker compose up`.
var Version = "dev"

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"version": Version,
	})
}
