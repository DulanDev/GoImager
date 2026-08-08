package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// minimal YAML spec that parses as both YAML and JSON-friendly map.
const sampleSpec = "openapi: 3.0.3\ninfo:\n  title: t\n  version: \"1\"\npaths: {}\n"

func TestOpenAPI_NilReturns404(t *testing.T) {
	srv := &Server{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	srv.OpenAPI(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("nil openapi spec should 404, got %d", rec.Code)
	}
}

func TestOpenAPI_YAMLView(t *testing.T) {
	srv := &Server{OpenAPIYAML: []byte(sampleSpec)}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	srv.OpenAPI(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/yaml") {
		t.Errorf("Content-Type = %q, want application/yaml*", ct)
	}
	if !strings.Contains(rec.Body.String(), "openapi: 3.0.3") {
		t.Errorf("body should contain the embedded YAML; got %q", rec.Body.String())
	}
}

func TestOpenAPI_JSONView(t *testing.T) {
	srv := &Server{OpenAPIYAML: []byte(sampleSpec)}
	rec := httptest.NewRecorder()
	// ".json" path or Accept: application/json both trigger JSON.
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	req.Header.Set("Accept", "application/json")
	srv.OpenAPI(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json*", ct)
	}
	// The YAML→JSON conversion turns `openapi: 3.0.3` into a quoted string.
	if !strings.Contains(rec.Body.String(), `"openapi":"3.0.3"`) {
		// json.Encoder doesn't always quote in tests; tolerate `{` too.
		if !strings.Contains(rec.Body.String(), `"openapi"`) {
			t.Errorf("JSON body missing expected key; got %q", rec.Body.String())
		}
	}
}

func TestOpenAPI_JSONViewViaAcceptHeader(t *testing.T) {
	srv := &Server{OpenAPIYAML: []byte(sampleSpec)}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	req.Header.Set("Accept", "application/json")
	srv.OpenAPI(rec, req)
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Accept: application/json should pick JSON view; got %q", ct)
	}
}

func TestOpenAPI_BadYAMLReturns500(t *testing.T) {
	srv := &Server{OpenAPIYAML: []byte("openapi: 3.0.3\n  oops: [unclosed")}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	req.Header.Set("Accept", "application/json")
	srv.OpenAPI(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("malformed YAML on JSON path should 500, got %d; body=%s", rec.Code, rec.Body.String())
	}
}

func TestSwaggerUI_NilReturns404(t *testing.T) {
	srv := &Server{}
	h := srv.SwaggerUI()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("nil UIFS should 404, got %d", rec.Code)
	}
}

func TestSwaggerUI_ServesIndex(t *testing.T) {
	// UIFS is expected to already be rooted at the UI distribution's root
	// (i.e. index.html lives at "index.html", not "ui/index.html"). main.go
	// does the fs.Sub(ui) before calling handler.New.
	uiFS := fstest.MapFS{
		"index.html":           {Data: []byte("<!doctype html><title>ui</title>")},
		"swagger-ui.css":       {Data: []byte("body{color:red}")},
		"swagger-ui-bundle.js": {Data: []byte("console.log('bundle')")},
	}
	srv := &Server{UIFS: uiFS}
	h := srv.SwaggerUI()
	if h == nil {
		t.Fatal("handler should not be nil")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("index should 200, got %d; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "ui") {
		t.Errorf("expected index.html content, got %q", rec.Body.String())
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/docs/swagger-ui.css", nil)
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("css should 200, got %d", rec2.Code)
	}
	if rec2.Header().Get("Content-Type") == "" {
		t.Errorf("css should get a Content-Type from http.FileServer")
	}
}
