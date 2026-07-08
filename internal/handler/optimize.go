package handler

import (
	"io"
	"net/http"
	"strconv"

	"github.com/DulanHewage/GoImager/internal/service"
)

// OptimizeHandler recompresses an uploaded image for web delivery. Required
// form field: image (file). Optional: quality (1-100, default 80),
// strip_exif (bool, default true — accomplished by decode-then-encode which
// drops metadata), format (jpeg|png|gif, default same as input).
func OptimizeHandler(w http.ResponseWriter, r *http.Request) {
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

	// Read the full upload so we can report original size and reuse the bytes.
	original, err := io.ReadAll(io.LimitReader(file, (maxFileSizeMB<<20)+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "READ_FAILED", "Unable to read uploaded image")
		return
	}
	if len(original) > maxFileSizeMB<<20 {
		writeError(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE",
			"Image exceeds max size of "+strconv.Itoa(maxFileSizeMB)+"MB")
		return
	}

	quality := 80
	if q := r.FormValue("quality"); q != "" {
		parsed, err := strconv.Atoi(q)
		if err != nil || parsed < 1 || parsed > 100 {
			writeError(w, http.StatusBadRequest, "INVALID_QUALITY", "quality must be an integer 1-100")
			return
		}
		quality = parsed
	}

	format := r.FormValue("format")
	if format != "" {
		if _, ok := normalizeFormat(format); !ok {
			writeError(w, http.StatusBadRequest, "INVALID_FORMAT", "Unsupported format: "+format)
			return
		}
	}

	result, err := service.OptimizeImage(original, format, quality)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "OPTIMIZE_FAILED", err.Error())
		return
	}

	w.Header().Set("Content-Type", result.ContentType)
	w.Header().Set("X-Original-Size", strconv.FormatInt(result.OriginalSize, 10))
	w.Header().Set("X-Optimized-Size", strconv.FormatInt(result.OptimizedSize, 10))
	reduction := 0.0
	if result.OriginalSize > 0 {
		reduction = (1 - float64(result.OptimizedSize)/float64(result.OriginalSize)) * 100
	}
	w.Header().Set("X-Reduction-Percent", strconv.FormatFloat(reduction, 'f', 2, 64))
	_, _ = w.Write(result.Bytes)
}