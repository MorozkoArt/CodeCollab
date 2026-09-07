# Подгружаем .env из корня монорепо
ifneq ("$(wildcard ../.env)","")
    include ../.env
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

ps:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) ps

# Отдельные сервисы
up-db:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) up --detach --build postgres

up-backend:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) up --detach --build backend

up-migrations:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) up --build migrations

up-nginx:
	$(DOCKER_COMPOSE) -f $(DEV_COMPOSE) up --detach --build nginx

# Prod
prod-up:
	$(DOCKER_COMPOSE) -f $(PROD_COMPOSE) pull
	$(DOCKER_COMPOSE) -f $(PROD_COMPOSE) up --detach --remove-orphans

prod-down:
	$(DOCKER_COMPOSE) -f $(PROD_COMPOSE) down --remove-orphans

prod-logs:
	$(DOCKER_COMPOSE) -f $(PROD_COMPOSE) logs -f

prod-ps:
	$(DOCKER_COMPOSE) -f $(PROD_COMPOSE) ps