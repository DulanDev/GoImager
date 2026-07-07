---
title: GoImager – Robust Self-Hosted Image Processing API Architecture
version: 1.0
date_created: 2026-07-07
last_updated: 2026-07-07
owner: DulanDev
tags: [architecture, api, image-processing, go, self-hosted, infrastructure]
---

# Introduction

This specification defines the target architecture, requirements, constraints, and interfaces for evolving **GoImager** from a minimal image resize/convert microservice into a robust, production-grade, self-hosted image processing API. The goal is feature parity with leading paid image APIs (Cloudinary, Imgix, ImageKit) while remaining fully self-hostable with a single Docker deployment.

## 1. Purpose & Scope

### Purpose

GoImager shall provide a comprehensive REST API that accepts images via upload or remote URL fetch, applies one or more chained transformations, and returns the result or stores it with a served URL.

### Scope

| In Scope | Out of Scope |
|---|---|
| Image transformation pipeline | Video processing |
| Format conversion and optimisation | Machine-learning model training |
| Storage backends (local, S3-compatible) | CDN edge distribution |
| Authentication and rate limiting | GUI/dashboard UI |
| URL-based transformation DSL | Cloud billing / usage metering |
| Async job queue for heavy transforms | Third-party DAM integration |

### Intended Audience

This document is intended for AI coding agents implementing GoImager features, human engineers extending or reviewing the codebase, and DevOps engineers deploying GoImager in production.

### Assumptions

- The runtime is Go 1.22+.
- Deployment target is Linux containers.
- The primary storage backend is local disk; S3-compatible object storage is secondary.
- Clients communicate over HTTP/1.1 or HTTP/2.

## 2. Definitions

| Term | Definition |
|---|---|
| API | Application Programming Interface |
| CDN | Content Delivery Network |
| DAM | Digital Asset Management |
| DSL | Domain-Specific Language used in transformation URLs |
| EXIF | Exchangeable Image File Format metadata |
| Fit mode | Strategy used when resizing changes aspect ratio |
| Gravity | Focal point used for cropping |
| Job | Asynchronous transformation task tracked by UUID |
| Origin URL | Remote URL of an image to fetch and process |
| Pipeline | Ordered sequence of transformations |
| REQ | Functional requirement |
| SEC | Security requirement |
| PER | Performance requirement |
| OPS | Operational requirement |
| CON | Constraint |
| GUD | Guideline |
| PAT | Pattern |

## 3. Requirements, Constraints & Guidelines

### 3.1 Functional Requirements

#### Image Input
- **REQ-001**: The API MUST accept image uploads via `multipart/form-data` POST requests.
- **REQ-002**: The API MUST accept a `url` parameter to fetch and process images from remote HTTP/HTTPS origins.
- **REQ-003**: The API MUST accept images via URL-based transformation DSL such as `GET /img/{operations}/{encoded-origin-url}`.
- **REQ-004**: The API MUST validate that uploaded or fetched content is a recognised image format before processing.
- **REQ-005**: Maximum accepted input file size MUST be configurable via `MAX_UPLOAD_BYTES`, default 50 MB.

#### Resize & Crop
- **REQ-010**: The API MUST support resizing by explicit width and/or height in pixels.
- **REQ-011**: The API MUST support fit modes: `contain`, `cover`, `fill`, `scale-down`, `crop`.
- **REQ-012**: The API MUST preserve aspect ratio when only one dimension is specified.
- **REQ-013**: The API MUST support cropping by explicit `x`, `y`, `width`, `height` coordinates.
- **REQ-014**: The API MUST support gravity-based cropping with values: `center`, `north`, `south`, `east`, `west`, `northeast`, `northwest`, `southeast`, `southwest`, `smart`.
- **REQ-015**: The API MUST support percentage-based dimensions.

#### Format Conversion & Optimisation
- **REQ-020**: The API MUST support output in `jpeg`, `png`, `webp`, `avif`, `gif`, `tiff`, `bmp`.
- **REQ-021**: The API MUST support quality control for lossy formats with integer range 1–100.
- **REQ-022**: The API MUST support `format=auto` based on the client `Accept` header.
- **REQ-023**: The API MUST support lossless compression mode for PNG and WebP.
- **REQ-024**: The API MUST strip EXIF metadata by default; `strip_exif=false` MUST allow retention.
- **REQ-025**: The API MUST support progressive JPEG encoding.

#### Image Transformations
- **REQ-030**: The API MUST support rotation by arbitrary degree with optional background fill.
- **REQ-031**: The API MUST support horizontal and vertical flip.
- **REQ-032**: The API MUST support brightness, contrast, and saturation adjustment in range -100 to +100.
- **REQ-033**: The API MUST support blur with configurable radius.
- **REQ-034**: The API MUST support sharpening.
- **REQ-035**: The API MUST support greyscale conversion.
- **REQ-036**: The API MUST support solid-colour or transparent padding/borders.
- **REQ-037**: The API MUST support watermarking with configurable position, size, and opacity.
- **REQ-038**: The API MUST support text overlay with configurable font, size, colour, opacity, and position.
- **REQ-039**: The API SHOULD support background removal through an external ML service or embedded model.
- **REQ-040**: The API SHOULD support super-resolution upscaling.

#### Pipeline / Chaining
- **REQ-050**: The API MUST support chaining multiple transformations in a single request, applied in order.
- **REQ-051**: Transformation chains MUST be expressible as both JSON body and URL query-parameter DSL.

#### Storage & Retrieval
- **REQ-060**: The API MUST support a `store=true` parameter that persists output to configured storage and returns a retrieval URL.
- **REQ-061**: The API MUST expose `GET /images/{id}` to retrieve stored images.
- **REQ-062**: The API MUST support local disk storage.
- **REQ-063**: The API SHOULD support S3-compatible object storage.
- **REQ-064**: Stored images MUST be retrievable via a stable URL with optional transformation parameters appended.
- **REQ-065**: The API MUST support deletion of stored images via `DELETE /images/{id}`.

#### Async Processing
- **REQ-070**: The API MUST support an async mode that returns a Job ID immediately and processes the image in the background.
- **REQ-071**: The API MUST expose `GET /jobs/{id}` to poll job status.
- **REQ-072**: Completed jobs MUST include the output image URL or inline base64 in the status response.

#### Metadata
- **REQ-080**: The API MUST expose `GET /info` accepting an image URL or upload and returning width, height, format, file size, colour mode, EXIF, and dominant colour.

### 3.2 Security Requirements

- **SEC-001**: All API endpoints MUST support API key authentication via `Authorization: Bearer {key}`.
- **SEC-002**: API keys MUST be configurable via `API_KEYS` or mounted secrets file.
- **SEC-003**: URL-fetch MUST enforce an allowlist of permitted origin domains via `ALLOWED_ORIGINS`.
- **SEC-004**: URL-fetch MUST reject requests to private IP ranges and loopback addresses.
- **SEC-005**: The API MUST enforce rate limiting per API key with default 60 requests/minute.
- **SEC-006**: Presigned URLs MUST expire after configurable TTL.
- **SEC-007**: The API MUST validate `Content-Type` and magic bytes of uploaded files.
- **SEC-008**: The API MUST enforce a maximum output pixel area through `MAX_OUTPUT_PIXELS`.
- **SEC-009**: URL-based DSL endpoints MUST support optional HMAC signature validation.

### 3.3 Performance Requirements

- **PER-001**: Resize and encode operations on images up to 5 MP MUST complete within 500 ms p99 under single-request load.
- **PER-002**: The API MUST support configurable worker concurrency.
- **PER-003**: The API MUST implement an in-memory LRU cache for transformation results.
- **PER-004**: The cache MUST support cache invalidation via `DELETE /cache/{key}`.

### 3.4 Operational Requirements

- **OPS-001**: The API MUST expose a `GET /health` endpoint returning HTTP 200 and service version.
- **OPS-002**: The API MUST expose a `GET /metrics` endpoint in Prometheus format.
- **OPS-003**: Metrics MUST include request count, latency histogram, error rate, cache hit/miss ratio, and active worker count.
- **OPS-004**: All logs MUST be structured JSON written to stdout.
- **OPS-005**: The API MUST support graceful shutdown with configurable drain timeout.

### 3.5 Constraints

- **CON-001**: The implementation language is Go version 1.22 or newer.
- **CON-002**: The service MUST be distributable as a single statically-linked binary and Docker image.
- **CON-003**: All configuration MUST be injectable via environment variables.
- **CON-004**: The API MUST remain backward-compatible unless a version increment is introduced.
- **CON-005**: Existing `/resize` and `/convert` endpoints MUST remain functional.

### 3.6 Guidelines

- **GUD-001**: Use `libvips` via Go bindings such as `govips` as the primary image processing backend.
- **GUD-002**: Adopt `chi` or `gin` for routing and middleware composition.
- **GUD-003**: Use table-driven unit tests for transformation functions.
- **GUD-004**: Document all public endpoints with OpenAPI 3.1.
- **GUD-005**: Use semantic versioning and expose version via `GET /health`.

### 3.7 Patterns

- **PAT-001**: Apply the Pipeline pattern for transformation chains.
- **PAT-002**: Apply the Strategy pattern for storage backends.
- **PAT-003**: Apply the Middleware pattern for authentication, rate limiting, logging, and recovery.
- **PAT-004**: Apply the Worker Pool pattern for async job processing.

## 4. Interfaces & Data Contracts

### 4.1 HTTP API Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/v1/process` | Process image and return or store result |
| `GET` | `/v1/img/{operations}/{encoded-url}` | URL-based DSL transformation |
| `GET` | `/v1/images/{id}` | Retrieve stored image |
| `DELETE` | `/v1/images/{id}` | Delete stored image |
| `GET` | `/v1/info` | Extract image metadata |
| `GET` | `/v1/jobs/{id}` | Poll async job status |
| `DELETE` | `/v1/cache/{key}` | Invalidate cache entry |
| `GET` | `/health` | Health check |
| `GET` | `/metrics` | Prometheus metrics |
| `POST` | `/resize` | Legacy resize endpoint |
| `POST` | `/convert` | Legacy convert endpoint |

### 4.2 `POST /v1/process` Request Schema

```jsonc
{
  "url": "https://example.com/photo.jpg",
  "format": "webp",
  "quality": 85,
  "lossless": false,
  "strip_exif": true,
  "progressive": false,
  "store": false,
  "async": false,
  "pipeline": [
    {
      "op": "resize",
      "width": 800,
      "height": 600,
      "fit": "cover",
      "gravity": "center"
    },
    {
      "op": "crop",
      "x": 0,
      "y": 0,
      "width": 400,
      "height": 400
    },
    { "op": "rotate", "angle": 90, "background": "#ffffff" },
    { "op": "flip", "direction": "horizontal" },
    { "op": "brightness", "value": 10 },
    { "op": "contrast", "value": -5 },
    { "op": "saturation", "value": 20 },
    { "op": "blur", "radius": 5 },
    { "op": "sharpen", "sigma": 1.0 },
    { "op": "greyscale" },
    { "op": "pad", "top": 10, "right": 10, "bottom": 10, "left": 10, "color": "#000000" },
    {
      "op": "watermark",
      "url": "https://example.com/logo.png",
      "position": "bottom-right",
      "opacity": 0.8,
      "width": 100
    },
    {
      "op": "text",
      "content": "© 2026 MyBrand",
      "font": "Roboto",
      "size": 24,
      "color": "#ffffff",
      "position": "bottom-left",
      "opacity": 0.9
    }
  ]
}
```

### 4.3 Response Schema

```jsonc
{
  "id": "img_01j9z3kx",
  "url": "https://yourdomain.com/v1/images/img_01j9z3kx",
  "format": "webp",
  "width": 800,
  "height": 600,
  "size_bytes": 34512,
  "created_at": "2026-07-07T12:00:00Z"
}
```

### 4.4 Job Status Schema

```jsonc
{
  "job_id": "job_01j9z3kx",
  "status": "done",
  "result": {
    "id": "img_01j9z3kx",
    "url": "/v1/images/img_01j9z3kx",
    "format": "webp",
    "width": 800,
    "height": 600,
    "size_bytes": 34512
  },
  "error": null,
  "created_at": "2026-07-07T12:00:00Z",
  "completed_at": "2026-07-07T12:00:01Z"
}
```

### 4.5 Info Response Schema

```jsonc
{
  "width": 3840,
  "height": 2160,
  "format": "jpeg",
  "size_bytes": 2048000,
  "color_mode": "sRGB",
  "has_alpha": false,
  "dominant_color": "#3a5f8b",
  "exif": {
    "make": "Fujifilm",
    "model": "X-T5",
    "focal_length": "35mm",
    "iso": 400,
    "aperture": "f/2.0",
    "shutter_speed": "1/250"
  }
}
```

### 4.6 URL-Based DSL

```text
GET /v1/img/rs:800:600:cover/q:85/f:webp/plain/https://example.com/photo.jpg
```

### 4.7 Error Response Schema

```jsonc
{
  "error": {
    "code": "INVALID_DIMENSIONS",
    "message": "Width must be between 1 and 10000 pixels",
    "request_id": "req_01j9z"
  }
}
```

### 4.8 Transformer Interface

```go
type Transformer interface {
    Transform(img any, op Op) error
}

type Op struct {
    Name   string
    Params map[string]any
}

type Pipeline struct {
    Steps []Op
}
```

### 4.9 StorageBackend Interface

```go
type StorageBackend interface {
    Put(ctx context.Context, key string, data []byte, contentType string) error
    Get(ctx context.Context, key string) ([]byte, string, error)
    Delete(ctx context.Context, key string) error
    URL(key string) string
}
```

### 4.10 Environment Variables

| Variable | Type | Default | Description |
|---|---|---|---|
| `PORT` | int | `8080` | HTTP server listen port |
| `MAX_UPLOAD_BYTES` | int | `52428800` | Max upload size |
| `MAX_OUTPUT_PIXELS` | int | `25000000` | Max output pixel area |
| `API_KEYS` | string | `` | Comma-separated valid API keys |
| `RATE_LIMIT_RPM` | int | `60` | Max requests per minute per API key |
| `WORKER_CONCURRENCY` | int | `NumCPU` | Async worker pool size |
| `CACHE_SIZE_MB` | int | `100` | In-memory LRU cache size |
| `STORAGE_BACKEND` | string | `local` | `local` or `s3` |
| `STORAGE_LOCAL_PATH` | string | `./data` | Local storage root |
| `ALLOWED_ORIGINS` | string | `*` | Allowlisted origin domains |
| `HMAC_KEY` | string | `` | Secret for DSL signing |
| `PRESIGNED_TTL_SECONDS` | int | `3600` | Presigned URL expiry |
| `SHUTDOWN_TIMEOUT_SECONDS` | int | `30` | Graceful shutdown timeout |
| `LOG_LEVEL` | string | `info` | Log verbosity |
| `BASE_URL` | string | `` | Public base URL |

## 5. Acceptance Criteria

- **AC-001**: Given a valid JPEG upload and resize pipeline, When `POST /v1/process` is called, Then the response is an image with expected dimensions.
- **AC-002**: Given `format=auto` and an `Accept` header preferring AVIF, When processing occurs, Then the response format is AVIF.
- **AC-003**: Given authentication is enabled and no bearer token is sent, When request is processed, Then the response is HTTP 401.
- **AC-004**: Given a private IP target in `url`, When processing is requested, Then the response is HTTP 422 with SSRF error.
- **AC-005**: Given `store=true`, When processing succeeds, Then the response contains a retrievable URL.
- **AC-006**: Given `async=true`, When processing is requested, Then the response is HTTP 202 with a Job ID.
- **AC-007**: Given a pipeline with resize then watermark, When applied, Then watermark positioning is relative to the transformed image.
- **AC-008**: Given `GET /v1/info`, When a valid image is provided, Then width, height, format, and file size are returned.
- **AC-009**: Given rate limiting is exceeded, When more requests arrive, Then the response is HTTP 429.
- **AC-010**: Given a valid legacy `/resize` request, When it is processed, Then it remains functional and returns HTTP 200.

## 6. Test Automation Strategy

- **Test Levels**: Unit, Integration, End-to-End.
- **Frameworks**: Go `testing`, `testify`, `httptest`, `golangci-lint`.
- **Test Data Management**: Use fixture images in `testdata/` covering standard, EXIF, transparent, invalid, and zero-byte files.
- **CI/CD Integration**: GitHub Actions must run tests, linting, and Docker image build on pull requests.
- **Coverage Requirements**: Minimum 70% statement coverage for core internal packages.
- **Performance Testing**: Benchmarks for resize, convert, and full pipeline execution.

## 7. Rationale & Context

The current GoImager implementation is limited to basic resize and format conversion. Paid platforms such as Cloudinary and Imgix differentiate themselves through composable pipelines, broad format support, URL-based transformations, automatic optimisation, and production-grade security features. This specification defines the minimal architecture required to close that gap while keeping deployment self-hosted and operationally simple.

The preferred image engine is `libvips` through Go bindings because it offers stronger performance and broader format support than pure-Go image libraries. A URL-based DSL and optional HMAC signing improve developer ergonomics and prevent unauthorised transformation abuse.

## 8. Dependencies & External Integrations

### External Systems
- **EXT-001**: Remote image origins over HTTP/HTTPS for URL fetch input.

### Third-Party Services
- **SVC-001**: S3-compatible object storage for persistent image storage.
- **SVC-002**: Optional ML-based background removal service.

### Infrastructure Dependencies
- **INF-001**: `libvips` with codec support for JPEG, PNG, WebP, AVIF, GIF, TIFF, and BMP.
- **INF-002**: Docker or Kubernetes runtime.

### Data Dependencies
- **DAT-001**: Font files for text overlays.

### Technology Platform Dependencies
- **PLT-001**: Go 1.22 or newer.
- **PLT-002**: Go bindings for `libvips`.

### Compliance Dependencies
- **COM-001**: GDPR/privacy constraints requiring EXIF stripping by default.

## 9. Examples & Edge Cases

```bash
curl -X POST https://imager.example.com/v1/process \
  -H "Authorization: Bearer my-api-key" \
  -F "url=https://cdn.mysite.com/original.jpg" \
  -F 'pipeline=[{"op":"resize","width":400,"height":400,"fit":"cover","gravity":"smart"},{"op":"watermark","url":"https://cdn.mysite.com/logo.png","position":"bottom-right","opacity":0.7}]' \
  -F "format=webp" \
  -F "quality=80" \
  --output thumbnail.webp
```

| Scenario | Expected Behaviour |
|---|---|
| `width=0, height=0` | Return HTTP 422 with `INVALID_DIMENSIONS` |
| Input file is 0 bytes | Return HTTP 422 with `EMPTY_FILE` |
| Input file is ZIP disguised as JPEG | Return HTTP 422 with `INVALID_IMAGE_FORMAT` |
| Output exceeds `MAX_OUTPUT_PIXELS` | Return HTTP 422 with `OUTPUT_TOO_LARGE` |
| Remote URL redirects to private IP | Return HTTP 422 with `SSRF_BLOCKED` |
| Watermark larger than base image | Scale watermark down to fit |
| Animated GIF input | Preserve animation across frames |

## 10. Validation Criteria

1. All `REQ-*` items have implementation and tests.
2. All `SEC-*` items have enforcement tests.
3. `POST /v1/process` supports all declared pipeline operations.
4. URL DSL processing works for documented token types.
5. `GET /v1/info` returns all documented metadata fields.
6. Health and metrics endpoints respond correctly.
7. Legacy endpoints remain functional.
8. Docker image builds and starts with defaults.
9. `go test ./... -race` passes.
10. `golangci-lint run` reports zero blocking issues.
11. Coverage meets minimum threshold.

## 11. Related Specifications / Further Reading

- [imgproxy Documentation – Processing Options](https://docs.imgproxy.net/usage/processing)
- [Imgix API Reference](https://docs.imgix.com/en-US/references/imgix-overview)
- [Cloudinary Image Transformation Reference](https://cloudinary.com/documentation/image_transformations)
- [libvips API Reference](https://www.libvips.org/API/current/)
- [OWASP SSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html)
