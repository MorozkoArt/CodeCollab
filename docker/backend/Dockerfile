FROM golang:1.25.4-alpine AS builder

RUN apk add --no-cache build-base

WORKDIR /app

COPY backend/go.mod backend/go.sum ./
COPY backend/vendor ./vendor
COPY backend/ .

ARG VERSION

RUN CGO_ENABLED=0 GOOS=linux \
    go build -mod=vendor \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o codecollab ./cmd/server/main.go

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/codecollab /app/codecollab

ENTRYPOINT ["/app/codecollab"]