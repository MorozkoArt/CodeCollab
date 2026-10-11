FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY . .

ARG VERSION=dev

RUN CGO_ENABLED=0 GOOS=linux \
    go build -mod=vendor \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /app/auth ./cmd/auth

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/auth /app/auth

ENTRYPOINT ["/app/auth"]