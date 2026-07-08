package service

import (
	"bytes"
	"fmt"
	"image"
	"io"

	"github.com/disintegration/imaging"
)

func ResizeImage(file io.Reader, width, height int) ([]byte, error) {
	// Read the image
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	// Resize the image
	resized := imaging.Resize(img, width, height, imaging.Lanczos)

	// Encode the resized image
	buf := new(bytes.Buffer)
	err = imaging.Encode(buf, resized, imaging.PNG)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func ConvertImage(file io.Reader, format string) ([]byte, string, error) {
	// Read the image
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, "", err
	}

	buf := new(bytes.Buffer)
	var contentType string

	// Encode the image in the target format
	switch format {
	case "jpeg":
		err = imaging.Encode(buf, img, imaging.JPEG)
		contentType = "image/jpeg"
	case "png":
		err = imaging.Encode(buf, img, imaging.PNG)
		contentType = "image/png"
	case "gif":
		err = imaging.Encode(buf, img, imaging.GIF)
		contentType = "image/gif"
	default:
		return nil, "", fmt.Errorf("unsupported format")
	}

	if err != nil {
		return nil, "", err
	}

	return buf.Bytes(), contentType, nil
}