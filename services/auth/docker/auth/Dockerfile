FROM golang:1.27-alpine AS builder

RUN apk add --no-cache build-base

WORKDIR /app

COPY pkg/go.mod pkg/go.sum ./pkg/
COPY pkg/ ./pkg/

COPY services/auth/go.mod services/auth/go.sum ./services/auth/
COPY services/auth/vendor ./services/auth/vendor
COPY services/auth/ ./services/auth/

WORKDIR /app/services/auth

ARG VERSION=dev

RUN CGO_ENABLED=0 GOOS=linux \
    go build -mod=vendor \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /app/auth ./cmd/server/main.go

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/auth /app/auth

ENTRYPOINT ["/app/auth"]