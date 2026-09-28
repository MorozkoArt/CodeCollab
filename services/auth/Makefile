# Подгружаем .env из корня монорепо
ifneq ("$(wildcard .env)","")
    include .env
    export
endif

LOCAL_BIN      := $(CURDIR)/bin
DOCKER_COMPOSE := docker compose
DEV_COMPOSE    := docker/dev/docker-compose.yml
PROD_COMPOSE   := docker/prod/docker-compose.yml
MIGRATIONS_DIR := db/migrations

ifeq ($(OS),Windows_NT)
    SET_GOBIN   := set "GOBIN=$(LOCAL_BIN)" &&
    NULL_DEVICE := NUL
else
    SET_GOBIN   := GOBIN=$(LOCAL_BIN)
    NULL_DEVICE := /dev/null
endif

# Проверяем доступность docker compose
ifeq (, $(shell $(DOCKER_COMPOSE) version 2>$(NULL_DEVICE)))
    DOCKER_COMPOSE := docker-compose
    ifeq (, $(shell $(DOCKER_COMPOSE) --version 2>$(NULL_DEVICE)))
        $(error "Neither 'docker compose' nor 'docker-compose' found in PATH")
    endif
endif

install-deps:
	$(SET_GOBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2
	$(SET_GOBIN) go install github.com/pressly/goose/v3/cmd/goose@v3.24.1
	$(SET_GOBIN) go install github.com/swaggo/swag/cmd/swag@latest
	$(SET_GOBIN) go install github.com/bufbuild/buf/cmd/buf@latest
	$(SET_GOBIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(SET_GOBIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Dev
up:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) up --detach --build --remove-orphans --force-recreate

down:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) down --remove-orphans

logs:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) logs -f

# Prod
prod-up:
	$(DOCKER_COMPOSE) -f $(PROD_COMPOSE) pull
	$(DOCKER_COMPOSE) -f $(PROD_COMPOSE) up --detach --remove-orphans

prod-down:
	$(DOCKER_COMPOSE) -f $(PROD_COMPOSE) down --remove-orphans

prod-logs:
	$(DOCKER_COMPOSE) -f $(PROD_COMPOSE) logs -f

# Миграции
m_create:
	$(LOCAL_BIN)/goose -dir $(MIGRATIONS_DIR) create $(m_name) sql

# Линтер
lint:
	$(LOCAL_BIN)/golangci-lint run --config=./.golangci.yml ./...

lint-fix:
	$(LOCAL_BIN)/golangci-lint run --config=./.golangci.yml --fix ./...

# Тесты
test:
	go test -cover -v -race ./internal/...

test-short:
	go test -short ./internal/...

test-cover:
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -html=coverage.out

# swagger
swagger:
	$(LOCAL_BIN)/swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

# Proto
proto:
	$(LOCAL_BIN)/buf generate