# CodeCollab Auth

The CodeCollab authentication service. It registers users, verifies credentials, and issues JWTs. The service provides an HTTP API for client applications and a gRPC API for internal services.

## Features

- user registration with unique `email` and `username`;
- login by email and password;
- password hashing before storing in PostgreSQL;
- JWT issuance using the HS256 algorithm;
- token validation and user profile retrieval via gRPC;
- Swagger UI and a health check.

## Requirements

- Docker Engine with Compose v2 (or `docker-compose`);
- Go **1.27+** — only for running without Docker, running tests, and generating artifacts;
- `grpcurl` — optional, for the gRPC examples below.

## Quick Start with Docker

From the monorepo root, create a `.env` file. The values below are suitable for local development. `SERVER_PORT=8081` is important when running through the bundled Nginx config: it proxies the service to port `8081`.

```dotenv
POSTGRES_USER=admin
POSTGRES_PASSWORD=change-me
POSTGRES_PORT=5432
POSTGRES_DB=codecollab
POSTGRES_HOST=postgres
POSTGRES_SSLMODE=disable

APP_ENV=development
SERVER_HOST=0.0.0.0
SERVER_PORT=8081
GRPC_PORT=9091
JWT_SECRET=replace-with-a-long-random-secret
JWT_EXPIRY=24h

NGINX_HTTP_PORT=80
NGINX_HTTPS_PORT=443
```

Start the entire local infrastructure from the repository root:

```bash
make up
```

This command brings up PostgreSQL, applies migrations, and starts auth and Nginx. To stop everything, run:

```bash
make down
```

Useful commands:

```bash
make logs-auth   # auth service logs
make logs-db     # PostgreSQL logs
make auth-test   # auth service tests
```

> When starting only auth (`make auth-up`), first create the Docker network and start the database: `make db-up`. Nginx is not required for this scenario.


To avoid loading `.env`, set `WITH_ENV_FILE=0` and pass all required environment variables to the process.

## Configuration

| Variable            | Default       | Description                                                                 |
|---------------------|---------------|-----------------------------------------------------------------------------|
| `APP_ENV`           | `development` | Environment used for application logging.                                   |
| `SERVER_HOST`       | `0.0.0.0`     | HTTP server address.                                                        |
| `SERVER_PORT`       | `8080`        | HTTP server port. For Nginx in this repository, use `8081`.                 |
| `GRPC_PORT`         | `9091`        | gRPC server port.                                                           |
| `JWT_SECRET`        | —             | JWT signing secret; be sure to set a secure value.                          |
| `JWT_EXPIRY`        | `24h`         | JWT lifetime in Go duration format, e.g. `12h` or `30m`.                    |
| `POSTGRES_HOST`     | `postgres`    | PostgreSQL host.                                                            |
| `POSTGRES_PORT`     | `5432`        | PostgreSQL port.                                                            |
| `POSTGRES_USER`     | `admin`       | Database user.                                                              |
| `POSTGRES_PASSWORD` | `11111111`    | Database password. Do not use this value outside local development.         |
| `POSTGRES_DB`       | `codecollab`  | Database name.                                                              |
| `POSTGRES_SSLMODE`  | `disable`     | SSL mode for the PostgreSQL connection.                                     |

## HTTP API

When connecting directly, replace `http://localhost:8081` with your actual `SERVER_HOST` and `SERVER_PORT`. When running through Nginx, the API is available at `https://auth.codecollab.local`; you will need a local DNS/hosts entry and a certificate from `nginx/ssl` that your system trusts.

All auth endpoint responses use the following envelope:

```json
{
  "success": true,
  "data": {}
}
```

On error, the `data` field is absent and the `error` field contains a description.

### Registration

`POST /api/v1/auth/register`

```bash
curl -i -X POST http://localhost:8081/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  --data '{"username":"alice","email":"alice@example.com","password":"correct-horse-battery-staple"}'
```

Expected successful response — `201 Created`:

```json
{"success":true}
```

Field constraints: `username` — 3 to 50 characters, `email` — a valid email address, `password` — at least 8 characters. A duplicate email returns `409 Conflict`.

### Login

`POST /api/v1/auth/login`

```bash
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  --data '{"email":"alice@example.com","password":"correct-horse-battery-staple"}'
```

A successful response (`200 OK`) contains the token:

```json
{
  "success": true,
  "data": {
    "token": "<jwt>"
  }
}
```

An incorrect email or password returns `401 Unauthorized`.

### Health Check

The service itself responds with JSON on `GET /api/v1/health`:

```bash
curl http://localhost:8081/api/v1/health
```

Swagger UI is available at `http://localhost:8081/swagger/` (or via the Nginx domain).

## gRPC API

The schema is located in [`api/proto/auth.proto`](./api/proto/auth.proto); the generated Go client is in [`pkg/authv1`](./pkg/authv1). The server has gRPC reflection enabled, so methods can be called with `grpcurl`:

```bash
# List available services
grpcurl -plaintext localhost:9091 list

# Registration
grpcurl -plaintext \
  -d '{"username":"bob","email":"bob@example.com","password":"correct-horse-battery-staple"}' \
  localhost:9091 auth.v1.AuthService/Register

# Login
grpcurl -plaintext \
  -d '{"email":"bob@example.com","password":"correct-horse-battery-staple"}' \
  localhost:9091 auth.v1.AuthService/Login

# JWT validation
grpcurl -plaintext -d '{"token":"<jwt>"}' \
  localhost:9091 auth.v1.AuthService/Validate

# Get user
grpcurl -plaintext -d '{"userId":"1"}' \
  localhost:9091 auth.v1.AuthService/GetUser
```

The `Register`, `Login`, `Validate`, and `GetUser` methods are described in the proto contract. gRPC returns `AlreadyExists` for an existing user, `Unauthenticated` for invalid credentials or a token, and `NotFound` if the user is not found.

## Development

Run commands from `services/auth`:

```bash
make test          # internal package tests with race detector and coverage
make test-short    # quick tests
make lint          # golangci-lint (after make install-deps)
make swagger       # regenerate docs/swagger.*
make proto         # generate Go code from proto via Buf
```

To install development tools, use `make install-deps`. To create a new SQL migration: `make m_create m_name=create_example_table`.

## Structure

```text
api/proto/              gRPC contract
cmd/server/             entry point
db/migrations/          PostgreSQL migrations (Goose)
internal/api/http/      HTTP handlers and middleware
internal/api/grpc/      gRPC handlers
internal/services/      business logic
internal/repo/          data access
pkg/authv1/             generated gRPC client and messages
```
