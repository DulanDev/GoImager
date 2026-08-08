# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.2.0] - 2026-08-07

### Security

- SSRF hardening on `GET /process`: outbound HTTP client now rejects
  private/reserved destinations (RFC 1918, link-local `169.254/16`
  including cloud metadata endpoints, loopback, IPv6 ULA/link-local,
  CGNAT, unspecified), preventing DNS-rebinding attacks. Redirects are
  re-validated against `ALLOWED_DOMAINS` on every hop, and fetch errors
  are masked to a generic message so resolved IPs/URLs cannot leak.
- `POST` endpoint API-key check switched to `crypto/subtle.ConstantTimeCompare`
  to remove a timing side-channel that could leak the key byte-by-byte.
- Server logs a loud `SECURITY: /process is open` warning at startup when
  `ALLOWED_DOMAINS="*"` is combined with no `SIGNING_KEY`, prompting the
  operator to tighten policy before public exposure.

### Changed

- Docker image now runs as a non-root user (`uid 10001`) and includes a
  `HEALTHCHECK` targeting `/health`. Bumped builder to `golang:1.23-alpine`
  to match `go.mod`. Added OCI image labels.
- `cmd/server/main.go` performs graceful shutdown on `SIGINT` / `SIGTERM`
  with a 30s drain via `http.Server.Shutdown`. In-flight uploads and
  `/process` fetches now complete instead of being killed mid-stream.
- Added `.dockerignore` so `COPY . .` no longer pulls the stale build
  binary, coverage artifacts, or `.git` into the Docker build context.

### Added

- Community files: `SECURITY.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`,
  this `CHANGELOG.md`.
- `internal/service/ssrf.go` — SSRF-aware dialer and HTTP client factory.

## [1.1.1] - 2024-08

### Added

- HMAC-signed URLs for `/process` (`SIGNING_KEY` / `SIGNING_KEYS`).
  Supports zero-downtime key rotation via `SIGNING_KEYS` (comma-separated,
  checked in order). Identical canonical form is reused by `cmd/gensig`
  for manual URL signing during local dev.
- OpenAPI spec bumped to `1.1.1` to reflect the signed-URL surfaces.

## [1.1.0] - 2024-07

### Added

- `GET /process` URL-parameter API: fetches a remote `src` image and
  applies resize / rotate / flip / blur / sharpen, then returns the
  transformed binary ready for direct `<img src>` use.
  Params: `src`, `w`, `h`, `mode`, `format`, `q`, `blur`, `sharp`,
  `rotate`, `flip`.
- `POST /thumbnail` — fixed-size center-crop via `imaging.Fill`.
- `API_KEY` bearer-token auth on POST endpoints (empty = open by design).
- Per-IP token-bucket rate limiting (`RATE_LIMIT_RPS` / `RATE_LIMIT_RPM`).
- `X-Processing-Time` response header.
- Bundled `libavif-apps` in the Docker image so native `avifenc`
  optimization works out of the box.

## [1.0.0] - 2024-07

### Added

- Initial public surface.
- `GET /health` liveness with version.
- `POST /info` — image metadata + EXIF (camera, taken_at, GPS) via an
  inline TIFF/EXIF parser with no external dependency.
- `POST /resize` — `fit` / `fill` / `stretch` modes; `0` = auto-scale
  on the omitted dimension; format passthrough when blank.
- `POST /convert` — JPEG / PNG / WebP / GIF / AVIF.
- `POST /optimize` — `pngquant` (PNG), `mozjpeg cjpeg` (JPEG via PPM
  intermediate), `cwebp` (WebP), `avifenc` (AVIF), with Go re-encode
  fallback when any tool is missing. `X-Original-Size`,
  `X-Optimized-Size`, `X-Reduction-Percent` response headers.
- Layered config: code defaults -> optional YAML (`./goimager.yaml`,
  `~/.goimager.yaml`, `/etc/goimager/config.yaml`) -> env vars.
- Structured `slog` request logging.
- Embedded OpenAPI 3.0.3 spec + offline Swagger UI (`/docs`,
  `/openapi.yaml`, `/openapi.json`).
- MIT license.

[Unreleased]: https://github.com/DulanDev/GoImager/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/DulanDev/GoImager/releases/tag/v1.2.0
[1.1.1]: https://github.com/DulanDev/GoImager/releases/tag/v1.1.1
[1.1.0]: https://github.com/DulanDev/GoImager/releases/tag/v1.1.0
[1.0.0]: https://github.com/DulanDev/GoImager/releases/tag/v1.0.0
