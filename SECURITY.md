# Security Policy

## Reporting a Vulnerability

Do **not** open a public GitHub issue. Use
[GitHub Private Vulnerability Reporting](https://github.com/DulanDev/GoImager/security/advisories/new)
(CVE issuance supported), or email **dulanhewage2@hotmail.com** if
unavailable. Include: affected version/commit, minimal reproduction,
impact, suggested fix. Acknowledgement within 5 business days;
fix-or-explanation target within 30 days.

## Threat Model

| Boundary                                                                   | Risk                                            | Mitigation                                                                                                                                                             |
| -------------------------------------------------------------------------- | ----------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| POST endpoints (`/resize`, `/convert`, `/optimize`, `/thumbnail`, `/info`) | Server-to-server over trusted network           | Bearer `API_KEY` gate (constant-time compare); rate-limited per IP. Empty key = open by design.                                                                        |
| `GET /process` (browser fetch of remote `src`)                             | Reached by browsers; secrets can't ship in HTML | Time-bound HMAC-signed URLs (`SIGNING_KEY[S]`); `ALLOWED_DOMAINS` allowlist; SSRF dial guard; redirect re-allowlist; masked fetch errors.                              |
| Remote `src` fetch targets                                                 | Untrusted — must not proxy internal networks    | Dialer rejects RFC 1918, link-local (`169.254.0.0/16` incl. cloud metadata), loopback, ULA, CGNAT, unspecified. DNS-rebinding to private range refused at dial time.   |
| Optimizer binaries (`pngquant`, `cjpeg`, `cwebp`, `avifenc`)               | Operator-controlled paths, fixed argv, no shell | User input never reaches argv. Missing tool → safe Go re-encode fallback, never a crash.                                                                               |
| Uploaded image bytes                                                       | Untrusted input                                 | stdlib `image.Decode` (EXIF dropped on decode). Body capped at `MAX_FILE_SIZE_MB`; source dims capped at `MAX_DIMENSION` / `MaxDimCap = 100000`. No external EXIF dep. |

## Safe defaults

- `/health`, `/`, `/openapi.*`, `/docs/*` always open (orchestrators +
  Swagger UI). Don't expose `/docs` to untrusted networks unless spec is
  meant public.
- `/process` **open by default** (no `SIGNING_KEY`). Startup logs
  `SECURITY: /process is open` when this coincides with
  `ALLOWED_DOMAINS="*"`. Tighten both before public exposure.
- Docker image runs non-root (`uid 10001`) with `HEALTHCHECK` on `/health`.

## Hardening checklist (public deployments)

1. `API_KEY` — strong random value for POST endpoints.
2. `SIGNING_KEY` (or `SIGNING_KEYS` for rotation) — sign every `/process`
   URL server-side. See `README.md` → _Authentication_ for Go/Node
   snippets; `cmd/gensig` for a CLI helper.
3. `ALLOWED_DOMAINS` — minimum set of source hosts, not `*`.
4. Reverse proxy terminating TLS + own rate limits / WAF.
5. Read-only filesystem; cap outbound egress at network level (in-process
   SSRF guard is defence-in-depth, not a substitute for egress filtering).
