package service

import (
	"bytes"
	"image"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"

	"github.com/DulanDev/GoImager/internal/config"
)

func TestThumbnail(t *testing.T) {
	out, ct, err := Thumbnail(bytes.NewReader(testPNG(t, 200, 100)), 50, 50, "png", 85, defaultCfg())
	if err != nil {
		t.Fatalf("thumbnail: %v", err)
	}
	if ct != "image/png" {
		t.Errorf("ct = %s", ct)
	}
	img, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 50 || b.Dy() != 50 {
		t.Errorf("dims = %dx%d, want 50x50", b.Dx(), b.Dy())
	}
}

func TestThumbnailInvalidDims(t *testing.T) {
	_, _, err := Thumbnail(bytes.NewReader(testPNG(t, 50, 50)), 0, 40, "png", 85, defaultCfg())
	ie, ok := err.(*ErrInvalid)
	if !ok || ie.Code != "INVALID_DIMENSIONS" {
		t.Fatalf("want INVALID_DIMENSIONS, got %T %v", err, err)
	}
}

func TestThumbnailDefaultFormatWebp(t *testing.T) {
	if _, err := exec.LookPath("cwebp"); err != nil {
		t.Skip("cwebp not installed")
	}
	out, _, err := Thumbnail(bytes.NewReader(testPNG(t, 60, 60)), 30, 30, "", 80, config.Optimizer{CwebpPath: "cwebp"})
	if err != nil {
		t.Fatalf("thumbnail webp: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("empty output")
	}
}

func TestAvifEncodeMissingTool(t *testing.T) {
	cfg := config.Optimizer{AvifPath: "/no/such/avifenc"}
	_, _, err := Encode(image.NewRGBA(image.Rect(0, 0, 8, 8)), "avif", 80, cfg)
	if err == nil {
		t.Fatal("expected error when avifenc missing")
	}
}

func TestNormalizeFormatAvif(t *testing.T) {
	f, err := NormalizeFormat("AVIF")
	if err != nil {
		t.Fatalf("normalize avif: %v", err)
	}
	if f != "avif" {
		t.Errorf("format = %s", f)
	}
	if ContentType("avif") != "image/avif" {
		t.Errorf("content type wrong")
	}
}

func TestProcessDomainBlocked(t *testing.T) {
	cfg := config.Config{Allowed: []string{"allowed.com"}}
	p := ProcessParams{Src: "https://evil.com/img.png"}
	_, _, err := Process(p, cfg, nil)
	ie, ok := err.(*ErrInvalid)
	if !ok || ie.Code != "DOMAIN_NOT_ALLOWED" {
		t.Fatalf("want DOMAIN_NOT_ALLOWED, got %T %v", err, err)
	}
}

func TestProcessMissingSrc(t *testing.T) {
	cfg := config.Config{Allowed: []string{"*"}}
	_, _, err := Process(ProcessParams{}, cfg, nil)
	if ie, ok := err.(*ErrInvalid); !ok || ie.Code != "MISSING_SRC" {
		t.Fatalf("want MISSING_SRC, got %T %v", err, err)
	}
}

func TestProcessHappyPath(t *testing.T) {
	pngBytes := testPNG(t, 200, 200)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngBytes)
	}))
	defer srv.Close()

	cfg := config.Config{Allowed: []string{"*"}}
	cfg.Server.MaxFileSizeMB = 20
	cfg.Quality.Default = 85
	p := ProcessParams{Src: srv.URL, W: 100, H: 100, Format: "png"}
	out, ct, err := Process(p, cfg, srv.Client())
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if ct != "image/png" {
		t.Errorf("ct = %s", ct)
	}
	img, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 100 || b.Dy() != 100 {
		t.Errorf("dims = %dx%d, want 100x100", b.Dx(), b.Dy())
	}
}

func TestProcessInvalidScheme(t *testing.T) {
	cfg := config.Config{Allowed: []string{"*"}}
	_, _, err := Process(ProcessParams{Src: "ftp://example.com/x.png"}, cfg, nil)
	ie, ok := err.(*ErrInvalid)
	if !ok || ie.Code != "INVALID_SRC" {
		t.Fatalf("want INVALID_SRC, got %T %v", err, err)
	}
}

func TestProcessNonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	cfg := config.Config{Allowed: []string{"*"}}
	_, _, err := Process(ProcessParams{Src: srv.URL}, cfg, srv.Client())
	if ie, ok := err.(*ErrInvalid); !ok || ie.Code != "SRC_FETCH_FAILED" {
		t.Fatalf("want SRC_FETCH_FAILED, got %T %v", err, err)
	}
}

func TestProcessTransformsBranch(t *testing.T) {
	pngBytes := testPNG(t, 200, 200)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(pngBytes)
	}))
	defer srv.Close()
	cfg := config.Config{Allowed: []string{"*"}}
	cfg.Server.MaxFileSizeMB = 20
	cfg.Quality.Default = 85
	cases := []struct {
		name string
		p    ProcessParams
	}{
		{"fill", ProcessParams{Src: srv.URL, W: 80, H: 80, Mode: "fill", Format: "png"}},
		{"stretch", ProcessParams{Src: srv.URL, W: 80, H: 40, Mode: "stretch", Format: "png"}},
		{"rotate180", ProcessParams{Src: srv.URL, Rotate: 180, Format: "png"}},
		{"flipH", ProcessParams{Src: srv.URL, Flip: "h", Format: "png"}},
		{"flipV", ProcessParams{Src: srv.URL, Flip: "v", Format: "png"}},
		{"blur", ProcessParams{Src: srv.URL, Blur: 1.5, Format: "png"}},
		{"sharp", ProcessParams{Src: srv.URL, Sharp: 1.0, Format: "png"}},
		{"rotate90", ProcessParams{Src: srv.URL, Rotate: 90, Format: "png"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, _, err := Process(c.p, cfg, srv.Client())
			if err != nil {
				t.Fatalf("process %s: %v", c.name, err)
			}
			if len(out) == 0 {
				t.Fatalf("empty output for %s", c.name)
			}
		})
	}
}

func TestProcessInvalidFormat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(testPNG(t, 40, 40))
	}))
	defer srv.Close()
	cfg := config.Config{Allowed: []string{"*"}}
	cfg.Server.MaxFileSizeMB = 20
	_, _, err := Process(ProcessParams{Src: srv.URL, Format: "bmp"}, cfg, srv.Client())
	if ie, ok := err.(*ErrInvalid); !ok || ie.Code != "INVALID_FORMAT" {
		t.Fatalf("want INVALID_FORMAT, got %T %v", err, err)
	}
}

func TestAvifEncodeReal(t *testing.T) {
	if _, err := exec.LookPath("avifenc"); err != nil {
		t.Skip("avifenc not installed")
	}
	cfg := config.Optimizer{AvifPath: "avifenc"}
	out, ct, err := Encode(image.NewRGBA(image.Rect(0, 0, 16, 16)), "avif", 80, cfg)
	if err != nil {
		t.Fatalf("avif encode: %v", err)
	}
	if ct != "image/avif" {
		t.Errorf("ct = %s", ct)
	}
	if len(out) == 0 {
		t.Fatal("empty avif output")
	}
}

func TestOptimizeAvifFallback(t *testing.T) {
	cfg := config.Optimizer{AvifPath: "/no/such/avifenc"}
	res, err := Optimize(bytes.NewReader(testPNG(t, 50, 50)), "avif", 80, true, cfg)
	if err != nil {
		t.Fatalf("optimize avif fallback: %v", err)
	}
	if res.ContentType != "image/jpeg" {
		t.Errorf("fallback ct = %s, want image/jpeg", res.ContentType)
	}
}
