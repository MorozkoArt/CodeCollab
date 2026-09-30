ifneq ("$(wildcard .env)","")
    include .env
    export
endif

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

DEV_COMPOSE  := dev/docker-compose.yml
PROD_COMPOSE := prod/docker-compose.yml

# Dev
up:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) up --detach --build --remove-orphans --force-recreate

down:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) down --remove-orphans

logs:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) logs -f