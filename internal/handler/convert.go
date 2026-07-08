package handler

import (
	"net/http"

	"github.com/DulanHewage/GoImager/internal/service"
)

// ConvertHandler converts an uploaded image to a target format. Required form
// fields: image (file), format (jpeg|png|gif).
func ConvertHandler(w http.ResponseWriter, r *http.Request) {
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

	format := r.FormValue("format")
	if _, ok := normalizeFormat(format); !ok {
		writeError(w, http.StatusBadRequest, "INVALID_FORMAT", "Unsupported format: "+format)
		return
	}

	converted, contentType, err := service.ConvertImage(file, format)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "CONVERT_FAILED", err.Error())
		return
	}

	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(converted)
}