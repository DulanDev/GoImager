# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
      -ldflags "-s -w -X github.com/DulanDev/GoImager/internal/handler.Version=${VERSION}" \
      -o goImager ./cmd/server

# Runtime stage
FROM alpine:3.19
RUN apk --no-cache add \
        ca-certificates \
        pngquant \
        libjpeg-turbo-utils \
        libwebp-tools \
        libavif-apps \
        wget \
    && adduser -D -u 10001 nonroot \
    && mkdir -p /app \
    && chown nonroot:nonroot /app

WORKDIR /app
USER nonroot:nonroot

COPY --from=builder /app/goImager /app/goImager

ENV PORT=8080 \
    LOG_FORMAT=json

EXPOSE 8080

LABEL org.opencontainers.image.source="https://github.com/DulanDev/GoImager" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.title="GoImager" \
      org.opencontainers.image.description="Self-hosted, privacy-first image processing microservice"

HEALTHCHECK --interval=30s --timeout=3s --start-period=2s --retries=3 \
    CMD wget -qO- "http://127.0.0.1:${PORT:-8080}/health" || exit 1

ENTRYPOINT ["/app/goImager"]