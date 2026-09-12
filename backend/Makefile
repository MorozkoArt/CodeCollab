# Подгружаем .env из корня монорепо
ifneq ("$(wildcard ../.env)","")
    include ../.env
    export
endif

LOCAL_BIN      := $(CURDIR)/bin
MIGRATIONS_DIR := db/migrations

GOOSE_DBSTRING_LOCAL := postgresql://$(DB_USER):$(DB_PASSWORD)@localhost:$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)

ifeq ($(OS),Windows_NT)
    SET_GOBIN    = set "GOBIN=$(LOCAL_BIN)" &&
    SET_TEST_DB  = set "TEST_DB_HOST=localhost" &&
else
    SET_GOBIN    = GOBIN=$(LOCAL_BIN)
    SET_TEST_DB  = TEST_DB_HOST=localhost
endif

install-deps:
	$(SET_GOBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2
	$(SET_GOBIN) go install github.com/pressly/goose/v3/cmd/goose@v3.24.1
	$(SET_GOBIN) go install github.com/swaggo/swag/cmd/swag@latest

# Vendor
vendor:
	go mod tidy
	go mod vendor

# Сборка
build:
	go build -mod=vendor \
		-ldflags="-s -w -X main.version=$(shell git rev-parse --short HEAD)" \
		-o bin/codecollab ./cmd/server/main.go

# Миграции
m_up:
	$(LOCAL_BIN)/goose -dir $(MIGRATIONS_DIR) postgres "$(GOOSE_DBSTRING_LOCAL)" up --allow-missing

m_down:
	$(LOCAL_BIN)/goose -dir $(MIGRATIONS_DIR) postgres "$(GOOSE_DBSTRING_LOCAL)" down

m_status:
	$(LOCAL_BIN)/goose -dir $(MIGRATIONS_DIR) postgres "$(GOOSE_DBSTRING_LOCAL)" status

m_create:
	$(LOCAL_BIN)/goose -dir $(MIGRATIONS_DIR) create $(m_name) sql

# Линтер
lint:
	$(LOCAL_BIN)/golangci-lint run --config=./.golangci.yml ./...

lint-fix:
	$(LOCAL_BIN)/golangci-lint run --config=./.golangci.yml --fix ./...

# Тесты
test:
	$(SET_TEST_DB) go test -cover -v -race ./internal/... ./pkg/...

test-short:
	$(SET_TEST_DB) go test -short ./internal/... ./pkg/...

test-cover:
	$(SET_TEST_DB) go test -coverprofile=coverage.out ./internal/... ./pkg/...
	go tool cover -html=coverage.out

# Генерация swagger документации
swagger:
	$(LOCAL_BIN)/swag init -g cmd/server/main.go -o docs