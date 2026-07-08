package service

import (
	"bytes"
	"image"
	"image/png"
	"os/exec"
	"strings"
	"testing"

	"github.com/DulanDev/GoImager/internal/config"
)

func testPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, img); err != nil {
		t.Fatalf("png encode: %v", err)
	}
	return buf.Bytes()
}

func defaultCfg() config.Optimizer {
	return config.Optimizer{PngquantPath: "pngquant", MozjpegPath: "cjpeg", CwebpPath: "cwebp"}
}

func TestResizeImageFit(t *testing.T) {
	out, ct, err := ResizeImage(bytes.NewReader(testPNG(t, 100, 100)), 50, 50, "fit", "png", 85, defaultCfg())
	if err != nil {
		t.Fatalf("resize: %v", err)
	}
	if ct != "image/png" {
		t.Errorf("content type = %s, want image/png", ct)
	}
	img, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode resized: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != 50 || b.Dy() != 50 {
		t.Errorf("dims = %dx%d, want 50x50", b.Dx(), b.Dy())
	}
}

func TestResizeImageFill(t *testing.T) {
	out, _, err := ResizeImage(bytes.NewReader(testPNG(t, 200, 100)), 50, 50, "fill", "png", 85, defaultCfg())
	if err != nil {
		t.Fatalf("fill resize: %v", err)
	}
	img, _, _ := image.Decode(bytes.NewReader(out))
	b := img.Bounds()
	if b.Dx() != 50 || b.Dy() != 50 {
		t.Errorf("fill dims = %dx%d, want 50x50", b.Dx(), b.Dy())
	}
}

func TestResizeImageStretch(t *testing.T) {
	out, _, err := ResizeImage(bytes.NewReader(testPNG(t, 100, 100)), 80, 30, "stretch", "jpeg", 85, defaultCfg())
	if err != nil {
		t.Fatalf("stretch resize: %v", err)
	}
	img, f, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if f != "jpeg" {
		t.Errorf("format = %s, want jpeg", f)
	}
	b := img.Bounds()
	if b.Dx() != 80 || b.Dy() != 30 {
		t.Errorf("stretch dims = %dx%d, want 80x30", b.Dx(), b.Dy())
	}
}

func TestResizeImageInvalid(t *testing.T) {
	cases := []struct {
		name          string
		w, h          int
		mode          string
		codeContains  string
	}{
		{"zero both", 0, 0, "fit", "INVALID_DIMENSIONS"},
		{"negative", -1, 50, "fit", "INVALID_DIMENSIONS"},
		{"bad mode", 50, 50, "bogus", "INVALID_MODE"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, err := ResizeImage(bytes.NewReader(testPNG(t, 40, 40)), c.w, c.h, c.mode, "png", 85, defaultCfg())
			ie, ok := err.(*ErrInvalid)
			if !ok {
				t.Fatalf("want ErrInvalid, got %T %v", err, err)
			}
			if !strings.Contains(ie.Code, c.codeContains) {
				t.Errorf("code = %s, want contains %s", ie.Code, c.codeContains)
			}
		})
	}
}

func TestConvertImage(t *testing.T) {
	out, ct, err := ConvertImage(bytes.NewReader(testPNG(t, 80, 60)), "jpeg", 90, defaultCfg())
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if ct != "image/jpeg" {
		t.Errorf("ct = %s", ct)
	}
	if _, f, err := image.Decode(bytes.NewReader(out)); err != nil || f != "jpeg" {
		t.Errorf("decoded format = %v err=%v", f, err)
	}
}

func TestConvertUnsupported(t *testing.T) {
	_, _, err := ConvertImage(bytes.NewReader(testPNG(t, 10, 10)), "bmp", 80, defaultCfg())
	if ie, ok := err.(*ErrInvalid); !ok || ie.Code != "INVALID_FORMAT" {
		t.Fatalf("want INVALID_FORMAT, got %T %v", err, err)
	}
}

func TestEncodeWebpFallback(t *testing.T) {
	cfg := config.Optimizer{CwebpPath: "/no/such/cwebp"}
	_, _, err := Encode(image.NewRGBA(image.Rect(0, 0, 8, 8)), "webp", 80, cfg)
	if err == nil {
		t.Fatal("expected error when cwebp missing")
	}
}

func TestOptimizeJPEG(t *testing.T) {
	if _, err := exec.LookPath("cjpeg"); err != nil {
		t.Skip("cjpeg not installed")
	}
	res, err := Optimize(bytes.NewReader(testPNG(t, 100, 100)), "jpeg", 80, true, defaultCfg())
	if err != nil {
		t.Fatalf("optimize jpeg: %v", err)
	}
	if res.ContentType != "image/jpeg" {
		t.Errorf("ct = %s", res.ContentType)
	}
}

func TestOptimizeFallbackNoTools(t *testing.T) {
	cfg := config.Optimizer{}
	res, err := Optimize(bytes.NewReader(testPNG(t, 60, 60)), "png", 80, true, cfg)
	if err != nil {
		t.Fatalf("optimize fallback: %v", err)
	}
	if res.ContentType != "image/png" {
		t.Errorf("ct = %s", res.ContentType)
	}
}

func TestInfoFromReader(t *testing.T) {
	info, err := InfoFromReader(bytes.NewReader(testPNG(t, 128, 96)))
	if err != nil {
		t.Fatalf("info: %v", err)
	}
	if info.Width != 128 || info.Height != 96 {
		t.Errorf("dims = %dx%d", info.Width, info.Height)
	}
	if info.Format != "png" {
		t.Errorf("format = %s", info.Format)
	}
	if info.SizeBytes <= 0 {
		t.Errorf("size = %d", info.SizeBytes)
	}
}