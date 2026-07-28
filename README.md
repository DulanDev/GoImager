# GoImager

[![Build](https://github.com/DulanDev/GoImager/actions/workflows/ci.yml/badge.svg)](https://github.com/DulanDev/GoImager/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A self-hosted, privacy-first image processing microservice written in Go.
Lightweight open-source alternative to Cloudinary/imgix - no SaaS fees, no
vendor lock-in, full data control.

## Features

- **Resize** with `fit`, `fill`, `stretch` modes
- **Convert** between JPEG / PNG / WebP / GIF / AVIF
- **Optimize** via optional external binaries (`pngquant`, `mozjpeg` `cjpeg`, `cwebp`, `avifenc`)
- **Info** — EXIF + image metadata extraction
- **Health** check for orchestrators
- **Process** — URL-parameter API with HMAC-signed URLs for direct `<img src>` usage (v1.1, signed v1.1.1)
- **Thumbnail** — fixed-size center-crop (v1.1)
- API key / bearer-token auth + per-IP rate limiting (v1.1)
- HMAC-signed URLs for `/process` (v1.1.1)
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
  "exif": {
    "camera": "Sony A7 IV",
    "taken_at": "2024-08-15T14:32:00Z",
    "gps": null
  }
}
```

### `POST /resize`

| Field     | Type    | Required | Description                             |
| --------- | ------- | -------- | --------------------------------------- |
| `image`   | file    | yes      | Image to resize                         |
| `width`   | integer | yes\*    | Target width in pixels. 0 = auto-scale  |
| `height`  | integer | yes\*    | Target height in pixels. 0 = auto-scale |
| `mode`    | string  | no       | `fit` (default), `fill`, `stretch`      |
| `format`  | string  | no       | Output: `jpeg`, `png`, `webp`, `gif`    |
| `quality` | integer | no       | Compression quality 1–100 (default: 85) |

\*At least one of `width` / `height` must be non-zero. When `format` is omitted
the input format is preserved.

Returns the transformed image binary with the matching `Content-Type`.

### `POST /convert`

| Field     | Type    | Required | Description                                  |
| --------- | ------- | -------- | -------------------------------------------- |
| `image`   | file    | yes      | Image to convert                             |
| `format`  | string  | yes      | Target: `jpeg`, `png`, `webp`, `gif`, `avif` |
| `quality` | integer | no       | Compression quality 1–100 (default: 85)      |

### `POST /optimize`

| Field        | Type    | Required | Description                              |
| ------------ | ------- | -------- | ---------------------------------------- |
| `image`      | file    | yes      | Image to optimize                        |
| `quality`    | integer | no       | Target quality 1–100 (default: 80)       |
| `strip_exif` | boolean | no       | Remove EXIF metadata (default: true)     |
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

| Param    | Type    | Description                                                                    |
| -------- | ------- | ------------------------------------------------------------------------------ |
| `src`    | string  | URL of the source image (URL-encoded)                                          |
| `w`      | integer | Target width                                                                   |
| `h`      | integer | Target height                                                                  |
| `mode`   | string  | `fit` (default), `fill`, `stretch`                                             |
| `format` | string  | `jpeg`, `png`, `webp`, `avif`                                                  |
| `q`      | integer | Quality 1–100                                                                  |
| `blur`   | float   | Gaussian blur radius (e.g. `1.5`)                                              |
| `sharp`  | float   | Sharpen sigma (e.g. `1.0`)                                                     |
| `rotate` | integer | `90`, `180`, `270`                                                             |
| `flip`   | string  | `h` (horizontal) or `v` (vertical)                                             |
| `exp`    | integer | _(signed mode)_ Expiry as Unix timestamp. Required when `SIGNING_KEY` is set.  |
| `sig`    | string  | _(signed mode)_ HMAC-SHA256 hex signature. Required when `SIGNING_KEY` is set. |

Example (unsigned, dev only — `SIGNING_KEY` unset):

```
GET /process?src=https%3A%2F%2Fexample.com%2Fphoto.jpg&w=800&format=webp&q=80
```

Example (signed):

```
GET /process?src=https%3A%2F%2Fexample.com%2Fphoto.jpg&w=800&format=webp&q=80&exp=1700000000&sig=5f2c...
```

The `src` host must appear in the `ALLOWED_DOMAINS` allowlist (`*` allows any).
When `SIGNING_KEY` is set, every `/process` URL must carry `exp` + `sig`;
expired → `410 Gone`, bad/missing signature → `401`. See
[Authentication](#authentication) for signing code.

### `POST /thumbnail` _(v1.1)_

Fixed-size center-crop thumbnail.

| Field     | Type    | Required | Description                     |
| --------- | ------- | -------- | ------------------------------- |
| `image`   | file    | yes      | Source image                    |
| `width`   | integer | yes      | Thumbnail width                 |
| `height`  | integer | yes      | Thumbnail height                |
| `format`  | string  | no       | Output format (default: `webp`) |
| `quality` | integer | no       | Quality 1–100 (default: 85)     |

## Authentication

GoImager supports **two independent auth mechanisms**, both **off by default**
so the service runs open on a trusted private network or behind a reverse
proxy. Enable either or both.

| Mechanism                                             | Protects                                                                   | When enabled                     | Failed request                     |
| ----------------------------------------------------- | -------------------------------------------------------------------------- | -------------------------------- | ---------------------------------- |
| **Bearer API key** (`API_KEY`)                        | POST endpoints (`/resize`, `/convert`, `/optimize`, `/thumbnail`, `/info`) | `API_KEY` env/YAML set non-empty | `401 UNAUTHORIZED`                 |
| **HMAC-signed URLs** (`SIGNING_KEY` / `SIGNING_KEYS`) | `GET /process` only                                                        | `SIGNING_KEY(S)` set non-empty   | `401 UNAUTHORIZED` / `410 EXPIRED` |

`/health` and `/` are always open. `/process` is exempt from the Bearer check
— its protection lives in signing. Both secrets default to empty = service
open (intended for private networks where the network _is_ the auth).

### Why two mechanisms

- **POST endpoints are called server-to-server** — your backend talks to
  GoImager over a private network. A static Bearer key in env is the right
  shape.
- **`/process` is browser-direct** — your backend renders
  `<img src="https://img.example.com/process?...">` tags that browsers load.
  A static secret shipped in HTML would leak. Signing keeps the secret
  server-side; only its (time-bound) signature travels to the client.

This mirrors imgproxy (`IMGPROXY_KEY`/`SALT` for signed URLs, plus a separate
`IMGPROXY_SECRET` for the Authorization header) and thumbor (HMAC in the URL
path, single `SECURITY_KEY`).

### Signing algorithm

1. Take the query string, **drop the `sig` parameter** (if present).
2. Sort remaining parameters by key lexicographically (stable; repeated keys
   sort by value). Encode each key/value as `application/x-www-form-urlencoded`
   (spaces are `+`, _not_ `%20` — matching Go's `net/url.Values.Encode()`);
   join with `&`.
3. The string to sign is `/process?` + that canonical query.
4. `sig = hex(HMAC_SHA256(key, string_to_sign))`.
5. Append `&sig=<sig>` to the URL. `exp` is a normal param and **must** be
   part of the signed string (it is not stripped — only `sig` is).

`SIGNING_KEYS` (comma-separated list) takes precedence over `SIGNING_KEY`
(single value; checked in order), enabling zero-downtime rotation: deploy
with `old,new`, switch clients to `new`, then redeploy with just `new`.

Server validation: missing `sig`/`exp` → `401`; `exp` in the past → `410`;
signature mismatch → `401` (no detail leaked). On success, normal
`ALLOWED_DOMAINS` enforcement applies to `src`.

> **CDN caveat:** headers aren't signed. CDNs must cache `/process` on the
> full URL (incl. `sig` + `exp`) to avoid bypassing expiry by header
> manipulation — same caveat as imgproxy.

### Sign in Go

```go
import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"time"
)

// signProcessURL appends exp + sig to q and returns the canonical /process
// query string. `key` must match the server's SIGNING_KEY or one of SIGNING_KEYS.
func signProcessURL(key string, q url.Values) string {
	q.Set("exp", strconv.FormatInt(time.Now().Add(24*time.Hour).Unix(), 10))
	q.Del("sig")
	canonical := q.Encode() // url.Values.Encode sorts keys and encodes spaces as '+'
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte("/process?" + canonical))
	q.Set("sig", hex.EncodeToString(mac.Sum(nil)))
	return q.Encode()
}
```

### Sign in JavaScript (Node)

`encodeURIComponent` alone encodes space as `%20`, but Go's canonical form
uses `+` — the `formEncode` helper below normalizes accordingly. Using this
helper is the difference between signatures that verify and ones that don't.

```js
const crypto = require("crypto");

function formEncode(s) {
  return encodeURIComponent(s).replace(/%20/g, "+");
}

function signProcessURL(key, params /*: URLSearchParams */) {
  const exp = String(Math.floor(Date.now() / 1000) + 86400); // 24h
  params.set("exp", exp);
  params.delete("sig");

  const entries = [...params.entries()].sort((a, b) =>
    a[0] < b[0] ? -1 : a[0] > b[0] ? 1 : 0,
  );
  const canonical = entries
    .map(([k, v]) => `${formEncode(k)}=${formEncode(v)}`)
    .join("&");

  const sig = crypto
    .createHmac("sha256", key)
    .update(`/process?${canonical}`)
    .digest("hex");
  params.set("sig", sig);
  return params.toString();
}
```

### Signs URLs for manual testing

The repo bundles a CLI generator that mirrors `internal/sign` exactly:

```sh
go run ./cmd/gensig -key=$SIGNING_KEY -exp=24h \
  -base=http://localhost:8080 \
  'src=https://example.com/photo.jpg&w=800&format=webp&q=80'
# -> http://localhost:8080/process?exp=...&format=webp&sig=...&src=...&w=800
```

Useful for `api.http` / curl during local dev; paste the output straight into
a request.

## Configuration

Precedence (low → high): code defaults → YAML file → environment variables.

YAML is searched in order: `./goimager.yaml`, `$HOME/.goimager.yaml`,
`/etc/goimager/config.yaml`. See [`goimager.example.yaml`](goimager.example.yaml).

| Variable                  | Default    | Description                                                                                                      |
| ------------------------- | ---------- | ---------------------------------------------------------------------------------------------------------------- |
| `PORT`                    | `8080`     | HTTP server port                                                                                                 |
| `MAX_FILE_SIZE_MB`        | `20`       | Max upload size in MB                                                                                            |
| `MAX_DIMENSION`           | `10000`    | Max allowed image dimension                                                                                      |
| `DEFAULT_QUALITY`         | `85`       | Default compression quality                                                                                      |
| `API_KEY`                 | _(empty)_  | If set, POST endpoints require `Authorization: Bearer <key>`. Empty = open (private-network mode).               |
| `SIGNING_KEY`             | _(empty)_  | If set, `/process` URLs must carry `sig` + `exp`. See [Authentication](#authentication).                         |
| `SIGNING_KEYS`            | _(unset)_  | Optional comma-separated list for zero-downtime rotation; checked in order. Takes precedence over `SIGNING_KEY`. |
| `RATE_LIMIT_RPS`          | `100`      | Requests per second per client IP                                                                                |
| `RATE_LIMIT_RPM`          | _(unset)_  | Per-minute limit; overrides RPS when set                                                                         |
| `ALLOWED_DOMAINS`         | `*`        | Comma-separated allowlist for `/process?src=`                                                                    |
| `LOG_LEVEL`               | `info`     | `debug` / `info` / `warn` / `error`                                                                              |
| `LOG_FORMAT`              | `json`     | `json` or `text`                                                                                                 |
| `OPTIMIZER_PNGQUANT_PATH` | `pngquant` | Path to `pngquant` binary (empty = skip)                                                                         |
| `OPTIMIZER_MOZJPEG_PATH`  | `cjpeg`    | Path to `mozjpeg` `cjpeg` binary (empty = skip)                                                                  |
| `OPTIMIZER_CWEBP_PATH`    | `cwebp`    | Path to `cwebp` binary (empty = skip)                                                                            |
| `OPTIMIZER_AVIF_PATH`     | `avifenc`  | Path to `avifenc` binary (empty = skip AVIF)                                                                     |

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
