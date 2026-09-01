# Подгружаем .env из корня
ifneq ("$(wildcard .env)","")
    include .env
    export
endif

DOCKER_COMPOSE := docker compose

# Проверяем доступность docker compose
ifeq (, $(shell $(DOCKER_COMPOSE) version 2> /dev/null))
    DOCKER_COMPOSE := docker-compose
    ifeq (, $(shell $(DOCKER_COMPOSE) --version 2> /dev/null))
        $(error "Neither 'docker compose' nor 'docker-compose' found in PATH")
    endif
endif

DEV_COMPOSE  := docker/dev/docker-compose.yml
PROD_COMPOSE := docker/prod/docker-compose.yml

# Dev
up:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) up --detach --build --remove-orphans --force-recreate

down:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) down --remove-orphans

logs:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) logs -f

ps:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) ps

# Отдельные сервисы
up-db:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) up --detach --build postgres

up-backend:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) up --detach --build backend

up-migrations:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) up --build migrations

# Prod
prod-up:
	$(DOCKER_COMPOSE) -f $(PROD_COMPOSE) pull
	$(DOCKER_COMPOSE) -f $(PROD_COMPOSE) up --detach --remove-orphans

prod-down:
	$(DOCKER_COMPOSE) -f $(PROD_COMPOSE) down --remove-orphans

prod-logs:
	$(DOCKER_COMPOSE) -f $(PROD_COMPOSE) logs -f

# backend/Makefile
lint:
	$(MAKE) -C backend lint

lint-fix:
	$(MAKE) -C backend lint-fix

test:
	$(MAKE) -C backend test

vendor:
	$(MAKE) -C backend vendor

install-deps:
	$(MAKE) -C backend install-deps

m_up:
	$(MAKE) -C backend m_up

m_down:
	$(MAKE) -C backend m_down

m_create:
	$(MAKE) -C backend m_create m_name=$(m_name)