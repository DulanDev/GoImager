package handler

import (
	"net/http"
	"strconv"

	"github.com/DulanDev/GoImager/internal/service"
)

func (s *Server) Convert(w http.ResponseWriter, r *http.Request) {
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

	format := r.FormValue("format")
	if format == "" {
		writeError(w, http.StatusBadRequest, "MISSING_FORMAT", "format field is required")
		return
	}

	quality := s.defaultQuality()
	if q := r.FormValue("quality"); q != "" {
		if n, err := strconv.Atoi(q); err == nil {
			quality = n
		} else {
			writeError(w, http.StatusBadRequest, "INVALID_QUALITY", "quality must be an integer 1-100")
			return
		}
	}

	out, ct, err := service.ConvertImage(file, format, quality, s.optimizerCfg())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", ct)
	w.Write(out)
}