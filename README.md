# GoImager

[![Build](https://github.com/DulanDev/GoImager/actions/workflows/ci.yml/badge.svg)](https://github.com/DulanDev/GoImager/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A self-hosted, privacy-first image processing microservice written in Go.
Lightweight open-source alternative to Cloudinary/imgix — no SaaS fees, no
vendor lock-in, full data control.

## Features

- **Resize** with `fit`, `fill`, `stretch` modes
- **Convert** between JPEG / PNG / WebP / GIF / AVIF
- **Optimize** via optional external binaries (`pngquant`, `mozjpeg` `cjpeg`, `cwebp`, `avifenc`)
- **Info** — EXIF + image metadata extraction
- **Health** check for orchestrators
- **Process** — URL-parameter API for direct `<img src>` usage (v1.1)
- **Thumbnail** — fixed-size center-crop (v1.1)
- API key / bearer-token auth + per-IP rate limiting (v1.1)
- `X-Processing-Time` response header (v1.1)
- Layered config: code defaults → optional YAML → env vars
- Structured `slog` request logging
- Stateless, horizontally scalable, Docker-ready

## Quick Start

```sh
git clone https://github.com/DulanDev/GoImager.git
cd GoImager
go mod tidy
go run cmd/server/main.go
```

Server listens on `http://localhost:8080` by default.

### Docker

```sh
docker compose up --build
```

The runtime image bundles `pngquant`, `mozjpeg` and `libwebp-tools`, so full
optimization works out of the box.

### Local optimizer tools (optional, for native dev)

```sh
brew install pngquant mozjpeg webp       # macOS
# or
sudo apt-get install -y pngquant mozjpeg webp   # Debian/Ubuntu
```

When a tool is missing the service logs a warning and falls back to Go's native
re-encode (metadata stripped, no quantization / Huffman optimization).

## API

All non-binary errors return structured JSON:

```json
{ "error": "description of what went wrong", "code": "INVALID_DIMENSIONS" }
```

### `GET /health`

```json
{ "status": "ok", "version": "1.0.0" }
```

### `GET|POST /info`

`multipart/form-data`, field `image` → metadata, no transformation.

```json
{
  "width": 1920,
  "height": 1080,
  "format": "jpeg",
  "size_bytes": 204800,
  "color_model": "YCbCr",
  "exif": { "camera": "Sony A7 IV", "taken_at": "2024-08-15T14:32:00Z", "gps": null }
}
```

### `POST /resize`

| Field     | Type    | Required | Description                                  |
| --------- | ------- | -------- | -------------------------------------------- |
| `image`   | file    | yes      | Image to resize                              |
| `width`   | integer | yes\*    | Target width in pixels. 0 = auto-scale       |
| `height`  | integer | yes\*    | Target height in pixels. 0 = auto-scale      |
| `mode`    | string  | no       | `fit` (default), `fill`, `stretch`           |
| `format`  | string  | no       | Output: `jpeg`, `png`, `webp`, `gif`         |
| `quality` | integer | no       | Compression quality 1–100 (default: 85)      |

\*At least one of `width` / `height` must be non-zero. When `format` is omitted
the input format is preserved.

Returns the transformed image binary with the matching `Content-Type`.

### `POST /convert`

| Field     | Type    | Required | Description                                         |
| --------- | ------- | -------- | --------------------------------------------------- |
| `image`   | file    | yes      | Image to convert                                    |
| `format`  | string  | yes      | Target: `jpeg`, `png`, `webp`, `gif`, `avif`        |
| `quality` | integer | no       | Compression quality 1–100 (default: 85)             |

### `POST /optimize`

| Field        | Type    | Required | Description                                     |
| ------------ | ------- | -------- | ----------------------------------------------- |
| `image`      | file    | yes      | Image to optimize                               |
| `quality`    | integer | no       | Target quality 1–100 (default: 80)              |
| `strip_exif` | boolean | no       | Remove EXIF metadata (default: true)            |
| `format`     | string  | no       | Output override (default: same as input) |

Headers: `X-Original-Size`, `X-Optimized-Size`, `X-Reduction-Percent`.

**Compression strategy per format:**

| Input format | Tool / method                       |
| ------------ | ----------------------------------- |
| PNG          | `pngquant` (24-bit → 8-bit indexed) |
| JPEG         | `mozjpeg` `cjpeg` (Huffman optim)   |
| WebP         | `cwebp`                             |
| GIF / other  | Go decode + re-encode fallback      |

All paths strip EXIF on decode.

### `GET /process` _(v1.1)_

URL-parameter API. Fetches a remote `src` image and returns the transformed
binary, ready for direct use in `<img src="...">`.

| Param    | Type    | Description                                  |
| -------- | ------- | -------------------------------------------- |
| `src`    | string  | URL of the source image (URL-encoded)        |
| `w`      | integer | Target width                                 |
| `h`      | integer | Target height                                |
| `mode`   | string  | `fit` (default), `fill`, `stretch`           |
| `format` | string  | `jpeg`, `png`, `webp`, `avif`                 |
| `q`      | integer | Quality 1–100                                |
| `blur`   | float   | Gaussian blur radius (e.g. `1.5`)            |
| `sharp`  | float   | Sharpen sigma (e.g. `1.0`)                   |
| `rotate` | integer | `90`, `180`, `270`                           |
| `flip`   | string  | `h` (horizontal) or `v` (vertical)           |

Example:

```
GET /process?src=https%3A%2F%2Fexample.com%2Fphoto.jpg&w=800&format=webp&q=80
```

The `src` host must appear in the `ALLOWED_DOMAINS` allowlist (`*` allows any).

### `POST /thumbnail` _(v1.1)_

Fixed-size center-crop thumbnail.

| Field    | Type    | Required | Description                     |
| -------- | ------- | -------- | ------------------------------- |
| `image`  | file    | yes      | Source image                    |
| `width`  | integer | yes      | Thumbnail width                 |
| `height` | integer | yes      | Thumbnail height                |
| `format` | string  | no       | Output format (default: `webp`) |
| `quality`| integer | no       | Quality 1–100 (default: 85)     |

## Configuration

Precedence (low → high): code defaults → YAML file → environment variables.

YAML is searched in order: `./goimager.yaml`, `$HOME/.goimager.yaml`,
`/etc/goimager/config.yaml`. See [`goimager.example.yaml`](goimager.example.yaml).

| Variable                     | Default     | Description                                            |
| ---------------------------- | ----------- | ------------------------------------------------------ |
| `PORT`                       | `8080`      | HTTP server port                                       |
| `MAX_FILE_SIZE_MB`           | `20`        | Max upload size in MB                                  |
| `MAX_DIMENSION`              | `10000`     | Max allowed image dimension                            |
| `DEFAULT_QUALITY`            | `85`        | Default compression quality                            |
| `API_KEY`                    | _(empty)_   | If set, requires `Authorization: Bearer <key>`         |
| `RATE_LIMIT_RPS`             | `100`       | Requests per second per client IP                      |
| `RATE_LIMIT_RPM`             | _(unset)_   | Per-minute limit; overrides RPS when set              |
| `ALLOWED_DOMAINS`            | `*`         | Comma-separated allowlist for `/process?src=`         |
| `LOG_LEVEL`                  | `info`      | `debug` / `info` / `warn` / `error`                    |
| `LOG_FORMAT`                 | `json`      | `json` or `text`                                       |
| `OPTIMIZER_PNGQUANT_PATH`    | `pngquant`  | Path to `pngquant` binary (empty = skip)               |
| `OPTIMIZER_MOZJPEG_PATH`     | `cjpeg`     | Path to `mozjpeg` `cjpeg` binary (empty = skip)        |
| `OPTIMIZER_CWEBP_PATH`       | `cwebp`     | Path to `cwebp` binary (empty = skip)                  |
| `OPTIMIZER_AVIF_PATH`        | `avifenc`   | Path to `avifenc` binary (empty = skip AVIF)           |

## Development

```sh
go mod tidy
go run cmd/server/main.go
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
go vet ./...
```

## License

MIT — see [LICENSE](LICENSE).