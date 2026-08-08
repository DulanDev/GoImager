package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"
)

// OpenAPI serves the embedded spec. When the request path ends in .json
// (or the Accept header prefers JSON), the YAML is unmarshalled and
// re-emitted as JSON for Swagger UI and other JSON-only consumers.
// Otherwise the YAML bytes are returned verbatim (Content-Type: application/yaml).
//
// Returns 404 when OpenAPIYAML is nil — the server's binary was built
// without the spec embedded (e.g. a downstream fork that dropped it).
func (s *Server) OpenAPI(w http.ResponseWriter, r *http.Request) {
	if len(s.OpenAPIYAML) == 0 {
		http.NotFound(w, r)
		return
	}
	wantJSON := strings.HasSuffix(r.URL.Path, ".json") ||
		strings.Contains(r.Header.Get("Accept"), "application/json")
	if !wantJSON {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(s.OpenAPIYAML)
		return
	}
	var doc interface{}
	if err := yaml.Unmarshal(s.OpenAPIYAML, &doc); err != nil {
		http.Error(w, "openapi: yaml parse: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(doc)
}

// SwaggerUI serves the vendored Swagger UI static assets rooted at /docs.
// `s.UIFS` is expected to already be Sub-d at construction time so its
// root contains index.html, swagger-ui.css, etc. directly. StripPrefix
// drops the /docs/ mount prefix before FileServer looks up the file.
func (s *Server) SwaggerUI() http.Handler {
	if s.UIFS == nil {
		return http.NotFoundHandler()
	}
	return http.StripPrefix("/docs", http.FileServer(http.FS(s.UIFS)))
}
