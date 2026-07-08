package handler

import (
	"encoding/json"
	"net/http"

	"github.com/DulanDev/GoImager/internal/service"
)

func (s *Server) Info(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(s.maxBytes()); err != nil {
		writeError(w, http.StatusBadRequest, "PAYLOAD_TOO_LARGE", "request body exceeds max file size")
		return
	}
	file, _, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "MISSING_IMAGE", "image field is required")
		return
	}
	defer file.Close()

	info, err := service.InfoFromReader(file)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(info)
}