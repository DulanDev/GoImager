package handler

import (
	"net/http"
	"strconv"

	"github.com/DulanDev/GoImager/internal/service"
)

func (s *Server) Optimize(w http.ResponseWriter, r *http.Request) {
	if !s.parseMultipartForm(w, r) {
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "MISSING_IMAGE", "image field is required")
		return
	}
	defer file.Close()

	originalSize := header.Size

	quality := 80
	if q := r.FormValue("quality"); q != "" {
		if n, err := strconv.Atoi(q); err == nil {
			quality = n
		} else {
			writeError(w, http.StatusBadRequest, "INVALID_QUALITY", "quality must be an integer 1-100")
			return
		}
	}

	stripExif := true
	if v := r.FormValue("strip_exif"); v == "false" || v == "0" {
		stripExif = false
	}

	format := r.FormValue("format")

	res, err := service.Optimize(file, format, quality, stripExif, s.optimizerCfg())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	optimizedSize := int64(len(res.Bytes))
	reduction := 0.0
	if originalSize > 0 {
		reduction = (1 - float64(optimizedSize)/float64(originalSize)) * 100
	}

	w.Header().Set("X-Original-Size", strconv.FormatInt(originalSize, 10))
	w.Header().Set("X-Optimized-Size", strconv.FormatInt(optimizedSize, 10))
	w.Header().Set("X-Reduction-Percent", strconv.FormatFloat(reduction, 'f', 2, 64))
	w.Header().Set("Content-Type", res.ContentType)
	_, _ = w.Write(res.Bytes)
}
