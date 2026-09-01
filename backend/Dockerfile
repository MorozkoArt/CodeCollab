FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags='-w -s' -o main ./cmd/server

RUN go install github.com/pressly/goose/v3/cmd/goose@latest

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY --from=builder /app/config.yaml .
COPY --from=builder /app/internal/db/migrations ./internal/db/migrations

RUN chown -R app:app /app

USER app

EXPOSE 8080

CMD ["./main"]