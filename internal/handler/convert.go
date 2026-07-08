package handler

import (
	"net/http"

	"GoImager/internal/service"
)

func ConvertHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB limit
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	// Get the file from the request
	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Unable to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get the target format
	format := r.FormValue("format")

	// Convert the image
	converted, contentType, err := service.ConvertImage(file, format)
	if err != nil {
		http.Error(w, "Unable to convert image", http.StatusInternalServerError)
		return
	}

	// Set the content type
	w.Header().Set("Content-Type", contentType)

	// Write the converted image to the response
	w.Write(converted)
}