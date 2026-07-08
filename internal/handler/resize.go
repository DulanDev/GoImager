package handler

import (
	"net/http"
	"strconv"

	"github.com/DulanHewage/GoImager/internal/service"
)

// ResizeHandler resizes an uploaded image. Required form fields:
//   image (file), width and/or height (int, 0 = auto-scale). Optional:
//   format (jpeg|png|gif, default png), quality (1-100, default 85).
func ResizeHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxFileSizeMB << 20); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_FORM", "Unable to parse multipart form")
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "MISSING_IMAGE", "Image file is required")
		return
	}
	defer file.Close()

	width, ok, msg := parseDimension(r.FormValue("width"), "width")
	if !ok {
		writeError(w, http.StatusBadRequest, "INVALID_DIMENSIONS", msg)
		return
	}
	height, ok, msg := parseDimension(r.FormValue("height"), "height")
	if !ok {
		writeError(w, http.StatusBadRequest, "INVALID_DIMENSIONS", msg)
		return
	}
	if width == 0 && height == 0 {
		writeError(w, http.StatusBadRequest, "INVALID_DIMENSIONS", "At least one of width or height must be non-zero")
		return
	}

	format := r.FormValue("format")
	if format == "" {
		format = "png"
	}
	if _, ok := normalizeFormat(format); !ok {
		writeError(w, http.StatusBadRequest, "INVALID_FORMAT", "Unsupported format: "+format)
		return
	}

	quality := defaultQuality
	if q := r.FormValue("quality"); q != "" {
		parsed, err := strconv.Atoi(q)
		if err != nil || parsed < 1 || parsed > 100 {
			writeError(w, http.StatusBadRequest, "INVALID_QUALITY", "quality must be an integer 1-100")
			return
		}
		quality = parsed
	}

	resized, contentType, err := service.ResizeImage(file, width, height, format, quality)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "RESIZE_FAILED", err.Error())
		return
	}

	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(resized)
}