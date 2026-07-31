# AGENTS.md

Compact guidance for working in this repo.

## Project

GoImager — self-hosted, privacy-first image processing microservice.
Module path: `github.com/DulanDev/GoImager`.

## Commands

```sh
go mod tidy                       # install/refresh deps
go run cmd/server/main.go         # start server (config: defaults -> goimager.yaml -> env)
go test ./...                     # run all tests
go test ./internal/service -run TestResizeImageFit   # run a single test by name
go test ./internal/service -coverprofile=coverage.out && go tool cover -func=coverage.out
go vet ./...                      # lint
docker compose up --build         # containerized run, bundles optimizer binaries
```

No Makefile, no scripts. `api/`, `pkg/`, `scripts/`, `config/` are empty
placeholder dirs — do not assume they hold code.

## Architecture

- `cmd/server/main.go` — only entrypoint; loads config via `config.Load()`,
  builds the `slog` logger, wires routes through `middleware.RequestLogger`,
  warns about missing optimizer tools.
- `internal/config/config.go` — layered config: code defaults → optional YAML
  (`./goimager.yaml`, `~/.goimager.yaml`, `/etc/goimager/config.yaml`) → env
  vars (highest priority). `.env` optionally loaded for local dev via
  `godotenv`, never fatal when absent.
- `internal/handler/` — handlers on `*Server`. `server.go` holds config + log +
  shared `writeError`. Endpoints: `health.go`(GET /health, constant Version),
  `info.go`(POST /info), `resize.go`, `convert.go`, `optimize.go`.
  All errors return structured JSON `{ "error", "code" }`.
- `internal/service/imageprocessor.go` — `ResizeImage` (modes fit/fill/stretch,
  dims validated, 0=auto, format passthrough when blank), `ConvertImage`,
  `Encode` (jpeg/png/gif via stdlib; webp via `cwebp` CLI), `Decode` (webp input
  registered via `golang.org/x/image/webp` blank import).
- `internal/service/optimizer.go` — `Optimize` dispatch: pngquant on PNG,
  mozjpeg `cjpeg` on JPEG (via PPM intermediate), `cwebp` on WebP; Go re-encode
  fallback when a tool/CLI is missing. `_ = stripExif` — EXIF is dropped by
  `image.Decode` automatically.
- `internal/service/metadata.go` — `InfoFromReader`; minimal inline TIFF/EXIF
  parser (no external EXIF dep) for Make/Model, DateTime(/Original) and GPS.
- `internal/middleware/logger.go` — `slog` request logger (`RequestLogger`
  adapter for mux `r.Use`).

## Conventions & gotchas

- Module path is **`github.com/DulanDev/GoImager`**. Imports are
  `github.com/DulanDev/GoImager/internal/...`. Renaming touches every file.
- No `.env` required. Config works with env vars only (Docker/K8s friendly).
- Multipart limit comes from `config.Server.MaxFileSizeMB` (default 20 MB).
- `MAX_DIMENSION` env/default `10000`; service also caps at internal
  `MaxDimCap = 100000` to reject oversized source images.
- WebP output requires `cwebp` on PATH (bundled in the Docker image). Without
  it, resize/convert/optimize with `format=webp` falls back to JPEG.
- `Optimizer` config CLI paths default to `pngquant` / `cjpeg` / `cwebp` /
  `avifenc`. Missing tools downgrade optimization but never crash the service.
- User-facing docs: `README.md` — API reference, quick start, config table.

## Changing the API

When you add, remove, or change a route (path, method, request/response
shape, auth), update **all** of these in the same changeset:

1. **Route registration** — `cmd/server/main.go` (the `HandleFunc` calls).
2. **OpenAPI spec** — `api/openapi.yaml` (path, schema, security). The
   Swagger UI (`/docs`) serves from this file.
3. **REST Client examples** — `api.http` request blocks.
4. **`README.md`** — API reference section and any prose that names the
   endpoint (e.g. the auth table).
5. **`internal/handler/*_test.go`** — any route registration that mirrors
   `main.go` (e.g. `v11_test.go`).
6. **`AGENTS.md`** — the handler summary line and any endpoint lists.

To find every mention before editing, grep the changed path across the
repo (e.g. `/info`) in `*.go`, `*.yaml`, `*.yml`, `*.md`, `*.http`,
`*.json`. Do not assume a path is mentioned only once.
