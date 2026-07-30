// Package api embeds the OpenAPI spec and the vendored Swagger UI assets so
// they ship inside the GoImager binary (no external file IO at runtime, no
// CDN dependency). Embed patterns are resolved relative to this package's
// directory, which is why this file lives next to openapi.yaml and the ui/
// subtree rather than under cmd/server.
package api

import "embed"

// OpenAPIYAML is the raw OpenAPI 3.0.3 spec served at /openapi.yaml and
// /openapi.json. Kept as bytes so the handler can return it verbatim for
// the YAML view and YAML->JSON-convert for the JSON view.
//
//go:embed openapi.yaml
var OpenAPIYAML []byte

// UIFS holds the Swagger UI static assets (index.html, swagger-ui-bundle.js,
// swagger-ui.css). The handler fs.Sub's this to root at "ui" before serving
// under /docs.
//
//go:embed ui
var UIFS embed.FS