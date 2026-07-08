package handler

import (
	"net/http"
	"strconv"

	"GoImager/internal/service"
)

func ResizeHandler(w http.ResponseWriter, r *http.Request) {
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

	// Get the new dimensions
	width, _ := strconv.Atoi(r.FormValue("width"))
	height, _ := strconv.Atoi(r.FormValue("height"))

	// Resize the image
	resized, err := service.ResizeImage(file, width, height)
	if err != nil {
		http.Error(w, "Unable to resize image", http.StatusInternalServerError)
		return
	}

	// Set the content type
	w.Header().Set("Content-Type", "image/png")

	// Write the resized image to the response
	w.Write(resized)
}