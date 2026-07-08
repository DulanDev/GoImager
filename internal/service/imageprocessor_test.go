package service

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

func TestResizeImage(t *testing.T) {
	// Create a test image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	buf := new(bytes.Buffer)
	png.Encode(buf, img)

	// Test resizing
	resized, err := ResizeImage(buf, 50, 50)
	if err != nil {
		t.Fatalf("Failed to resize image: %v", err)
	}

	// Decode the resized image
	decodedImg, _, err := image.Decode(bytes.NewReader(resized))
	if err != nil {
		t.Fatalf("Failed to decode resized image: %v", err)
	}

	// Check dimensions
	bounds := decodedImg.Bounds()
	if bounds.Dx() != 50 || bounds.Dy() != 50 {
		t.Errorf("Expected dimensions 50x50, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestConvertImage(t *testing.T) {
	// Create a test image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	buf := new(bytes.Buffer)
	png.Encode(buf, img)

	// Test conversion to JPEG
	converted, contentType, err := ConvertImage(buf, "jpeg")
	if err != nil {
		t.Fatalf("Failed to convert image: %v", err)
	}

	if contentType != "image/jpeg" {
		t.Errorf("Expected content type image/jpeg, got %s", contentType)
	}

	// Attempt to decode as JPEG
	_, format, err := image.DecodeConfig(bytes.NewReader(converted))
	if err != nil {
		t.Fatalf("Failed to decode converted image: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("Expected format jpeg, got %s", format)
	}
}