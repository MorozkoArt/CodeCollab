DOCKER_COMPOSE := docker compose

ifeq ($(OS),Windows_NT)
    NULL_DEVICE := NUL
else
    NULL_DEVICE := /dev/null
endif

# Проверяем доступность docker compose
ifeq (, $(shell $(DOCKER_COMPOSE) version 2>$(NULL_DEVICE)))
    DOCKER_COMPOSE := docker-compose
    ifeq (, $(shell $(DOCKER_COMPOSE) --version 2>$(NULL_DEVICE)))
        $(error "Neither 'docker compose' nor 'docker-compose' found in PATH")
    endif
endif

# Dev: поднять всё последовательно
up:
	$(MAKE) -C database up
	$(MAKE) -C services/auth up
	$(MAKE) -C nginx up

down:
	$(MAKE) -C nginx down
	$(MAKE) -C services/auth down
	$(MAKE) -C database down

logs-db:
	$(MAKE) -C database logs

logs-auth:
	$(MAKE) -C services/auth logs

logs-nginx:
	$(MAKE) -C nginx logs

# Отдельные сервисы
db-up:
	$(MAKE) -C database up

db-down:
	$(MAKE) -C database down

auth-up:
	$(MAKE) -C services/auth up

auth-down:
	$(MAKE) -C services/auth down

auth-test:
	$(MAKE) -C services/auth test

auth-lint:
	$(MAKE) -C services/auth lint

auth-migrate:
	$(MAKE) -C services/auth m_up

nginx-up:
	$(MAKE) -C nginx up

nginx-down:
	$(MAKE) -C nginx down

# Prod
prod-up:
	$(MAKE) -C database prod-up
	$(MAKE) -C services/auth prod-up
	$(MAKE) -C nginx prod-up

prod-down:
	$(MAKE) -C nginx prod-down
	$(MAKE) -C services/auth prod-down
	$(MAKE) -C database prod-down

# Разработка
lint-all:
	$(MAKE) -C services/auth lint

test-all:
	$(MAKE) -C services/auth test