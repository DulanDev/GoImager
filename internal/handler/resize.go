package handler

import (
	"net/http"
	"strconv"

	"github.com/DulanDev/GoImager/internal/service"
)

func (s *Server) Resize(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusBadRequest, "INVALID_DIMENSIONS", "width must be an integer")
		return
	}
	height, err := strconv.Atoi(r.FormValue("height"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_DIMENSIONS", "height must be an integer")
		return
	}

	mode := r.FormValue("mode")
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

	out, ct, err := service.ResizeImage(file, width, height, mode, format, quality, s.optimizerCfg())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", ct)
	_, _ = w.Write(out)
}

func writeServiceError(w http.ResponseWriter, err error) {
	if inv, ok := err.(*service.ErrInvalid); ok {
		writeError(w, http.StatusBadRequest, inv.Code, inv.Message)
		return
	}
	writeError(w, http.StatusInternalServerError, "INTERNAL", err.Error())
}
