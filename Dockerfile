# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o goImager ./cmd/server

# Runtime stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates pngquant mozjpeg libwebp-tools libavif-apps
WORKDIR /root/
COPY --from=builder /app/goImager .
EXPOSE 8080
CMD ["./goImager"]