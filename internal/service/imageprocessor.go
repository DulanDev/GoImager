package service

import (
	"bytes"
	"fmt"
	"image"
	"io"

	"github.com/disintegration/imaging"

	_ "golang.org/x/image/bmp"  // bmp input support
	_ "golang.org/x/image/tiff" // tiff input support
	_ "golang.org/x/image/webp" // webp input support
)

const defaultQuality = 85

// ResizeImage decodes an image, resizes it to width x height (0 = preserve
// aspect ratio), and re-encodes it in the requested format with the given
// quality. An empty format defaults to png (lossless).
func ResizeImage(file io.Reader, width, height int, format string, quality int) ([]byte, string, error) {
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, "", fmt.Errorf("unable to decode image: %w", err)
	}

	if width == 0 && height == 0 {
		return nil, "", fmt.Errorf("width and height cannot both be 0")
	}

	resized := imaging.Resize(img, width, height, imaging.Lanczos)

	encFormat, contentType, err := resolveFormat(format)
	if err != nil {
		return nil, "", err
	}

	buf := new(bytes.Buffer)
	opts := encodeOpts(encFormat, quality)
	if err := imaging.Encode(buf, resized, encFormat, opts...); err != nil {
		return nil, "", fmt.Errorf("unable to encode resized image: %w", err)
	}
	return buf.Bytes(), contentType, nil
}

// ConvertImage decodes an image and re-encodes it in the requested format.
func ConvertImage(file io.Reader, format string) ([]byte, string, error) {
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, "", fmt.Errorf("unable to decode image: %w", err)
	}

	encFormat, contentType, err := resolveFormat(format)
	if err != nil {
		return nil, "", err
	}

	buf := new(bytes.Buffer)
	if err := imaging.Encode(buf, img, encFormat); err != nil {
		return nil, "", fmt.Errorf("unable to encode converted image: %w", err)
	}
	return buf.Bytes(), contentType, nil
}

// OptimizeImageResult holds optimization output bytes plus before/after sizes.
type OptimizeImageResult struct {
	Bytes        []byte
	ContentType  string
	OriginalSize int64
	OptimizedSize int64
}

// OptimizeImage decodes an image, drops its metadata by re-encoding (EXIF
// stripping via decode-then-encode), and recompresses it at the requested
// quality. An empty format preserves the source format when detectable,
// otherwise falls back to jpeg for lossy size reduction.
func OptimizeImage(file []byte, format string, quality int) (*OptimizeImageResult, error) {
	img, srcFormat, err := image.Decode(bytes.NewReader(file))
	if err != nil {
		return nil, fmt.Errorf("unable to decode image: %w", err)
	}

	if format == "" {
		format = srcFormat
		if format == "" {
			format = "jpeg"
		}
	}

	encFormat, contentType, err := resolveFormat(format)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	opts := encodeOpts(encFormat, quality)
	if err := imaging.Encode(buf, img, encFormat, opts...); err != nil {
		return nil, fmt.Errorf("unable to optimize image: %w", err)
	}

	return &OptimizeImageResult{
		Bytes:         buf.Bytes(),
		ContentType:   contentType,
		OriginalSize:  int64(len(file)),
		OptimizedSize: int64(buf.Len()),
	}, nil
}

// resolveFormat maps a request format name to an imaging.Format plus HTTP
// content type.
func resolveFormat(format string) (imaging.Format, string, error) {
	switch format {
	case "jpeg", "jpg":
		return imaging.JPEG, "image/jpeg", nil
	case "png":
		return imaging.PNG, "image/png", nil
	case "gif":
		return imaging.GIF, "image/gif", nil
	default:
		return 0, "", fmt.Errorf("unsupported format: %q", format)
	}
}

// encodeOpts returns imaging.EncodeOptions appropriate for a format + quality.
func encodeOpts(f imaging.Format, quality int) []imaging.EncodeOption {
	if f == imaging.JPEG && (quality <= 0 || quality > 100) {
		quality = defaultQuality
	}
	if f != imaging.JPEG {
		return nil
	}
	return []imaging.EncodeOption{imaging.JPEGQuality(quality)}
}

