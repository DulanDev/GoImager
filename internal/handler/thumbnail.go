package handler

import (
	"net/http"
	"strconv"

	"github.com/DulanDev/GoImager/internal/service"
)

func (s *Server) Thumbnail(w http.ResponseWriter, r *http.Request) {
	if !s.parseMultipartForm(w, r) {
		return
	}
	file, _, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "MISSING_IMAGE", "image field is required")
		return
	}
	defer file.Close()

	width, err := strconv.Atoi(r.FormValue("width"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_DIMENSIONS", "width must be a positive integer")
		return
	}
	height, err := strconv.Atoi(r.FormValue("height"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_DIMENSIONS", "height must be a positive integer")
		return
	}

	format := r.FormValue("format")
	quality := s.defaultQuality()
	if q := r.FormValue("quality"); q != "" {
		if n, err := strconv.Atoi(q); err == nil {
			quality = n
		} else {
			writeError(w, http.StatusBadRequest, "INVALID_QUALITY", "quality must be an integer 1-100")
			return
		}
	}

	out, ct, err := service.Thumbnail(file, width, height, format, quality, s.optimizerCfg())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", ct)
	w.Write(out)
}
