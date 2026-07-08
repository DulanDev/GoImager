package service

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/DulanDev/GoImager/internal/config"
	_ "golang.org/x/image/webp"
)

const MaxDimCap = 100000

var ErrUnsupportedFormat = errors.New("unsupported format")

func SupportedFormats() []string {
	return []string{"jpeg", "png", "webp", "gif"}
}

func NormalizeFormat(f string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(f)) {
	case "", "jpeg", "jpg":
		return "jpeg", nil
	case "png":
		return "png", nil
	case "webp":
		return "webp", nil
	case "gif":
		return "gif", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedFormat, f)
	}
}

func ContentType(format string) string {
	switch format {
	case "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	case "gif":
		return "image/gif"
	}
	return "application/octet-stream"
}

type ErrInvalid struct {
	Code    string
	Message string
}

func (e *ErrInvalid) Error() string { return e.Message }

func Decode(src io.Reader) (image.Image, string, error) {
	var buf bytes.Buffer
	img, format, err := image.Decode(io.TeeReader(src, &buf))
	if err != nil {
		return nil, "", err
	}
	return img, format, nil
}

func ResizeImage(src io.Reader, width, height int, mode, format string, quality int, cfg config.Optimizer) ([]byte, string, error) {
	if width < 0 || height < 0 {
		return nil, "", &ErrInvalid{Code: "INVALID_DIMENSIONS", Message: "width and height must be >= 0"}
	}
	if width == 0 && height == 0 {
		return nil, "", &ErrInvalid{Code: "INVALID_DIMENSIONS", Message: "at least one of width or height must be non-zero"}
	}
	quality = clampQuality(quality)

	img, inFormat, err := Decode(src)
	if err != nil {
		return nil, "", &ErrInvalid{Code: "INVALID_IMAGE", Message: fmt.Sprintf("could not decode image: %v", err)}
	}

	fmtArg := format
	if strings.TrimSpace(fmtArg) == "" {
		fmtArg = inFormat
	}
	if fmtArg, err = NormalizeFormat(fmtArg); err != nil {
		return nil, "", &ErrInvalid{Code: "INVALID_FORMAT", Message: err.Error()}
	}

	bounds := img.Bounds()
	if bounds.Dx() > MaxDimCap || bounds.Dy() > MaxDimCap {
		return nil, "", &ErrInvalid{Code: "INVALID_DIMENSIONS", Message: "source image exceeds max dimension"}
	}

	if width > 0 {
		width = min(width, MaxDimCap)
	}
	if height > 0 {
		height = min(height, MaxDimCap)
	}

	var out image.Image
	m := strings.ToLower(strings.TrimSpace(mode))
	switch m {
	case "", "fit":
		out = imaging.Fit(img, width, height, imaging.Lanczos)
	case "fill":
		out = imaging.Fill(img, width, height, imaging.Center, imaging.Lanczos)
	case "stretch":
		out = imaging.Resize(img, width, height, imaging.Lanczos)
	default:
		return nil, "", &ErrInvalid{Code: "INVALID_MODE", Message: "mode must be fit, fill or stretch"}
	}

	return Encode(out, fmtArg, quality, cfg)
}

func ConvertImage(src io.Reader, format string, quality int, cfg config.Optimizer) ([]byte, string, error) {
	fmtArg, err := NormalizeFormat(format)
	if err != nil {
		return nil, "", &ErrInvalid{Code: "INVALID_FORMAT", Message: err.Error()}
	}
	quality = clampQuality(quality)

	img, _, err := Decode(src)
	if err != nil {
		return nil, "", &ErrInvalid{Code: "INVALID_IMAGE", Message: fmt.Sprintf("could not decode image: %v", err)}
	}
	return Encode(img, fmtArg, quality, cfg)
}

func Encode(img image.Image, format string, quality int, cfg config.Optimizer) ([]byte, string, error) {
	buf := new(bytes.Buffer)
	switch format {
	case "jpeg":
		if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, "", err
		}
	case "png":
		enc := png.Encoder{CompressionLevel: png.BestCompression}
		if err := enc.Encode(buf, img); err != nil {
			return nil, "", err
		}
	case "gif":
		if err := gif.Encode(buf, img, &gif.Options{NumColors: 256}); err != nil {
			return nil, "", err
		}
	case "webp":
		return encodeWebp(img, quality, cfg)
	default:
		return nil, "", fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}
	return buf.Bytes(), ContentType(format), nil
}

func encodeWebp(img image.Image, quality int, cfg config.Optimizer) ([]byte, string, error) {
	if cfg.CwebpPath == "" {
		return nil, "", &ErrInvalid{Code: "WEBP_UNAVAILABLE", Message: "cwebp not configured"}
	}
	pngBuf := new(bytes.Buffer)
	if err := png.Encode(pngBuf, img); err != nil {
		return nil, "", err
	}
	cmd := exec.Command(cfg.CwebpPath, "-quiet", "-q", strconv.Itoa(quality), "-o", "-", "--", "-")
	cmd.Stdin = pngBuf
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, "", fmt.Errorf("cwebp failed: %w: %s", err, errBuf.String())
	}
	return out.Bytes(), ContentType("webp"), nil
}

func clampQuality(q int) int {
	if q <= 0 || q > 100 {
		return 85
	}
	return q
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}