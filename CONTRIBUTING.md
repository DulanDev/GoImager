# Contributing to GoImager

Thanks for considering a contribution. GoImager is a small, focused
microservice — keep PRs scoped and you'll merge fast.

See [`AGENTS.md`](AGENTS.md) for the project layout and architecture
overview.

## Setup

```sh
git clone https://github.com/DulanDev/GoImager.git
cd GoImager
go mod tidy
go run cmd/server/main.go            # http://localhost:8080
```

Optional optimizer binaries (only needed for full optimization):

```sh
brew install pngquant mozjpeg webp           # macOS
sudo apt-get install -y pngquant mozjpeg webp libavif-bin  # Debian/Ubuntu
```

Missing tools log a warning and fall back to a safe Go re-encode — the
service never crashes.

## Before you push

```sh
go vet ./...
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

Coverage should not regress. The CI floor isn't enforced yet; behave as
if it is.

If you touch the OpenAPI spec:

```sh
npx @stoplight/spectral-cli@6.14.1 lint api/openapi.yaml
```

Configuration lives in `.spectral.yaml`.

## Changing the API

Grep the changed path across `*.go`, `*.yaml`, `*.yml`, `*.md`, `*.http`,
`*.json` before editing — do not assume a path is mentioned only once. See
[`AGENTS.md`](AGENTS.md) for the full checklist of files to update per route
change.

## Code style

- Idiomatic Go. Run `gofmt -s` before committing.
- Errors returned to clients are the structured `{"error", "code"}`
  shape — see `internal/handler/server.go` `writeError`. Add a new
  `code` constant only when the failure mode is meaningfully distinct.
- Service-level errors use `&service.ErrInvalid{Code, Message}` so the
  handler can map them via `writeServiceError`.
- No `log.Println` in handlers/services — use the `*slog.Logger` from the
  `*Server`. Keep secrets out of log fields.
- No new dependencies without justification in the PR description.
  `gorilla/mux` is legacy; new code should avoid leaning on more mux
  features than the standard `ServeMux` can express.

## Commit messages

Conventional Commits:

```
feat: add /grayscale endpoint
fix: clamp dimension to MaxDimCap before Decode
docs: document AVIF in the optimize table
refactor: extract applyTransforms into its own file
chore: bump golang.org/x/image
test: cover /convert happy path
build: add golangci-lint to CI
```

Keep commits atomic and small. One logical change per commit.

## Pull requests

- Targeted, one feature/fix per PR.
- Title matches the commit prefix (`feat:`, `fix:`, etc.).
- Describe what changed and **why**. Call out any security implications
  (auth, SSRF, parsing) explicitly.
- Add or update tests in the same PR. New endpoint → new handler test at
  minimum.
- Update `CHANGELOG.md` under an `## Unreleased` section.

## Security

Found a security issue? See [`SECURITY.md`](SECURITY.md). Do **not**
open a public issue.

## Code of Conduct

Participation in this project is governed by the
[Contributor Covenant 2.1](CODE_OF_CONDUCT.md). Be excellent to each
other.