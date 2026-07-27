package service

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"os/exec"
	"strconv"

	"github.com/DulanDev/GoImager/internal/config"
)

type OptimizeResult struct {
	Bytes       []byte
	ContentType string
}

const ppmHeaderTemplate = "P6\n%d %d\n255\n"

func writePPM(img image.Image) []byte {
	bounds := img.Bounds()
	var buf bytes.Buffer
	fmt.Fprintf(&buf, ppmHeaderTemplate, bounds.Dx(), bounds.Dy())
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			buf.Write([]byte{byte(r >> 8), byte(g >> 8), byte(b >> 8)})
		}
	}
	return buf.Bytes()
}

func Optimize(src io.Reader, format string, quality int, stripExif bool, cfg config.Optimizer) (*OptimizeResult, error) {
	fmtArg, err := NormalizeFormat(format)
	if err != nil {
		return nil, &ErrInvalid{Code: "INVALID_FORMAT", Message: err.Error()}
	}
	quality = clampQuality(quality)

	img, inFormat, err := Decode(src)
	if err != nil {
		return nil, &ErrInvalid{Code: "INVALID_IMAGE", Message: fmt.Sprintf("could not decode image: %v", err)}
	}
	_ = stripExif

	outFormat := fmtArg

	switch outFormat {
	case "png":
		if pngquantAvail(cfg) {
			data, err := goEncodePNG(img)
			if err != nil {
				return nil, err
			}
			out, err := runPngquant(data, quality, cfg)
			if err != nil {
				return &OptimizeResult{Bytes: data, ContentType: ContentType("png")}, nil
			}
			return &OptimizeResult{Bytes: out, ContentType: ContentType("png")}, nil
		}
		data, err := goEncodePNG(img)
		if err != nil {
			return nil, err
		}
		return &OptimizeResult{Bytes: data, ContentType: ContentType("png")}, nil

	case "jpeg":
		if mozjpegAvail(cfg) {
			ppm := writePPM(img)
			out, err := runCjpeg(ppm, quality, cfg)
			if err != nil {
				return fallbackJPEG(img, quality, inFormat)
			}
			return &OptimizeResult{Bytes: out, ContentType: ContentType("jpeg")}, nil
		}
		return fallbackJPEG(img, quality, inFormat)

	case "webp":
		if cfg.CwebpPath != "" && toolExists(cfg.CwebpPath) {
			out, ct, err := Encode(img, "webp", quality, cfg)
			if err == nil {
				return &OptimizeResult{Bytes: out, ContentType: ct}, nil
			}
		}
		return fallbackJPEG(img, quality, inFormat)

	case "gif":
		data, ct, err := Encode(img, "gif", quality, cfg)
		if err != nil {
			return nil, err
		}
		return &OptimizeResult{Bytes: data, ContentType: ct}, nil

	case "avif":
		if cfg.AvifPath != "" && toolExists(cfg.AvifPath) {
			out, ct, err := Encode(img, "avif", quality, cfg)
			if err == nil {
				return &OptimizeResult{Bytes: out, ContentType: ct}, nil
			}
		}
		return fallbackJPEG(img, quality, inFormat)
	}
	return nil, fmt.Errorf("optimize: unreachable format %q", outFormat)
}

func goEncodePNG(img image.Image) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func fallbackJPEG(img image.Image, quality int, inFormat string) (*OptimizeResult, error) {
	out, ct, err := Encode(img, "jpeg", quality, config.Optimizer{})
	if err != nil {
		return nil, err
	}
	return &OptimizeResult{Bytes: out, ContentType: ct}, nil
}

func runPngquant(pngData []byte, quality int, cfg config.Optimizer) ([]byte, error) {
	if !toolExists(cfg.PngquantPath) {
		return nil, fmt.Errorf("pngquant not found")
	}
	minQ, maxQ := pngQualityRange(quality)
	args := []string{"--quality=" + strconv.Itoa(minQ) + "-" + strconv.Itoa(maxQ), "--strip", "-"}
	cmd := exec.Command(cfg.PngquantPath, args...)
	cmd.Stdin = bytes.NewReader(pngData)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pngquant failed: %w: %s", err, errBuf.String())
	}
	return out.Bytes(), nil
}

func runCjpeg(ppm []byte, quality int, cfg config.Optimizer) ([]byte, error) {
	if !toolExists(cfg.MozjpegPath) {
		return nil, fmt.Errorf("mozjpeg cjpeg not found")
	}
	cmd := exec.Command(cfg.MozjpegPath, "-quality", strconv.Itoa(quality), "-optimize", "-progressive")
	cmd.Stdin = bytes.NewReader(ppm)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cjpeg failed: %w: %s", err, errBuf.String())
	}
	return out.Bytes(), nil
}

func pngQualityRange(q int) (int, int) {
	minQ := q - 10
	maxQ := q + 5
	if minQ < 0 {
		minQ = 0
	}
	if maxQ > 100 {
		maxQ = 100
	}
	return minQ, maxQ
}

func toolExists(path string) bool {
	if path == "" {
		return false
	}
	if p, err := exec.LookPath(path); err == nil && p != "" {
		return true
	}
	return false
}

func pngquantAvail(cfg config.Optimizer) bool {
	return toolExists(cfg.PngquantPath)
}

func mozjpegAvail(cfg config.Optimizer) bool {
	return toolExists(cfg.MozjpegPath)
}

