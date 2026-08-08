package handler

import (
	"bytes"
	"image"
	"image/png"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DulanDev/GoImager/internal/config"
	"github.com/gorilla/mux"
)

func newTestServer(cfg config.Config) *Server {
	return &Server{
		Cfg: cfg,
		Log: slog.Default(),
	}
}

func rawPNGBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func pngBody(t *testing.T, w, h int) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, err := mw.CreateFormFile("image", "t.png")
	if err != nil {
		t.Fatal(err)
	}
	fw.Write(rawPNGBytes(t, w, h))
	mw.Close()
	return body, mw.FormDataContentType()
}

// multipartWithFields returns a multipart body containing the image
// plus one form field per map entry.
func multipartWithFields(t *testing.T, imgBytes []byte, fields map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, err := mw.CreateFormFile("image", "t.png")
	if err != nil {
		t.Fatal(err)
	}
	fw.Write(imgBytes)
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	mw.Close()
	return body, mw.FormDataContentType()
}

func TestThumbnailHandler(t *testing.T) {
	srv := newTestServer(config.Config{Allowed: []string{"*"}})
	body, ct := multipartWithFields(t, rawPNGBytes(t, 200, 200), map[string]string{"width": "50", "height": "50", "format": "png"})
	req := httptest.NewRequest(http.MethodPost, "/thumbnail", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	srv.Thumbnail(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "image/png" {
		t.Errorf("ct = %s", rec.Header().Get("Content-Type"))
	}
}

func TestThumbnailHandlerMissingDims(t *testing.T) {
	srv := newTestServer(config.Config{})
	body, ct := pngBody(t, 50, 50)
	req := httptest.NewRequest(http.MethodPost, "/thumbnail", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	srv.Thumbnail(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "INVALID_DIMENSIONS") {
		t.Errorf("body = %s", rec.Body.String())
	}
}

func TestProcessHandlerDomainBlocked(t *testing.T) {
	srv := newTestServer(config.Config{Allowed: []string{"allowed.com"}})
	req := httptest.NewRequest(http.MethodGet, "/process?src=https://evil.com/x.png", nil)
	rec := httptest.NewRecorder()
	srv.Process(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "DOMAIN_NOT_ALLOWED") {
		t.Errorf("body = %s", rec.Body.String())
	}
}

func TestAuthMiddlewareBlocksRoute(t *testing.T) {
	srv := newTestServer(config.Config{Auth: config.Auth{APIKey: "k"}})
	r := mux.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		// simulate AuthAdapter for route test
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if h := req.Header.Get("Authorization"); h != "Bearer k" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, req)
		})
	})
	r.HandleFunc("/info", srv.Info).Methods("POST")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/info", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}
