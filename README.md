# Команды для запуска

## Запуск Docker-compose

```bash
docker compose up --build
```

## Применение миграций

```bash
docker exec -it codecollab-backend sh
goose -dir internal/db/migrations up
```

## Для входа в psql

```bash
docker exec -it codecollab-db psql -U app_user -d codecollab
\dt # Посмотреть все таблицы
Select * FROM users; # Посмотреть всех пользователей в таблице
```
