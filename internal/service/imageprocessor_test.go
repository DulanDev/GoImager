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

	// Test resizing to png (default lossless format)
	resized, contentType, err := ResizeImage(buf, 50, 50, "png", 0)
	if err != nil {
		t.Fatalf("Failed to resize image: %v", err)
	}
	if contentType != "image/png" {
		t.Errorf("Expected content type image/png, got %s", contentType)
	}

	decodedImg, _, err := image.Decode(bytes.NewReader(resized))
	if err != nil {
		t.Fatalf("Failed to decode resized image: %v", err)
	}

	bounds := decodedImg.Bounds()
	if bounds.Dx() != 50 || bounds.Dy() != 50 {
		t.Errorf("Expected dimensions 50x50, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestResizeImageRejectsZeroDimensions(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	buf := new(bytes.Buffer)
	png.Encode(buf, img)

	if _, _, err := ResizeImage(buf, 0, 0, "png", 0); err == nil {
		t.Fatal("Expected error when both width and height are 0")
	}
}

func TestResizeImageRejectsUnsupportedFormat(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	buf := new(bytes.Buffer)
	png.Encode(buf, img)

	if _, _, err := ResizeImage(buf, 50, 50, "avif", 0); err == nil {
		t.Fatal("Expected error for unsupported format avif")
	}
}

func TestConvertImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	buf := new(bytes.Buffer)
	png.Encode(buf, img)

	converted, contentType, err := ConvertImage(buf, "jpeg")
	if err != nil {
		t.Fatalf("Failed to convert image: %v", err)
	}

	if contentType != "image/jpeg" {
		t.Errorf("Expected content type image/jpeg, got %s", contentType)
	}

	_, format, err := image.DecodeConfig(bytes.NewReader(converted))
	if err != nil {
		t.Fatalf("Failed to decode converted image: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("Expected format jpeg, got %s", format)
	}
}

func TestOptimizeImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	buf := new(bytes.Buffer)
	png.Encode(buf, img)
	src := buf.Bytes()

	result, err := OptimizeImage(src, "jpeg", 75)
	if err != nil {
		t.Fatalf("Failed to optimize image: %v", err)
	}
	if result.ContentType != "image/jpeg" {
		t.Errorf("Expected content type image/jpeg, got %s", result.ContentType)
	}
	if result.OriginalSize != int64(len(src)) {
		t.Errorf("Expected original size %d, got %d", len(src), result.OriginalSize)
	}
	if result.OptimizedSize <= 0 {
		t.Errorf("Expected non-zero optimized size, got %d", result.OptimizedSize)
	}

	if _, _, err := image.Decode(bytes.NewReader(result.Bytes)); err != nil {
		t.Fatalf("Failed to decode optimized image: %v", err)
	}
}