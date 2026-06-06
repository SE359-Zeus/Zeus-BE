# Installing the System Service

The **System** service is the identity and auth backbone of Zeus. It manages users, roles, JWT token signing, audit logs, and email notifications.

- **Port:** 8085
- **Database:** SQLite
- **Image:** `ghcr.io/se359-zeus/zeus-be/system:latest`
- **Special role:** Holds the JWT **private key** -- all other services use the corresponding public key to verify tokens.

---

## Prerequisites

### Bare Metal
- **Go 1.26+**
- **GCC / build-base** (CGO required for `mattn/go-sqlite3`)
- **SQLite3 dev headers**
- **Valkey** or **Redis** (optional, for caching sessions and RBAC)
- **RabbitMQ** (optional, for consuming audit events)
- **SMTP server** (optional, for account creation emails)

### Docker
- **Docker 24+** with Compose V2

---

## Configuration

System reads a `.env` file via `godotenv` or environment variables directly.

```env
# ── Required ────────────────────────────────────────────────────────────────
SERVER_PORT=8085
DB_PATH=system.db

# ── JWT key pair ────────────────────────────────────────────────────────────
# System uses the PRIVATE key to sign tokens.
# All other services use the PUBLIC key to verify them.
JWT_PRIVATE_KEY_PATH=jwt_private.pem
JWT_PUBLIC_KEY_PATH=jwt_public.pem

# ── Optional: cache ─────────────────────────────────────────────────────────
VALKEY_ADDR=redis://localhost:6379

# ── Optional: message broker ────────────────────────────────────────────────
RABBITMQ_URL=

# ── Optional: email (SMTP) ──────────────────────────────────────────────────
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=
SMTP_PASS=
EMAIL_FROM_ADDRESS=no-reply@example.com
EMAIL_FROM_NAME="Zeus"
EMAIL_TEMPLATE_DIR=./templates

# ── Optional: public URL ────────────────────────────────────────────────────
PUBLIC_BASE_URL=http://localhost:8085

# ── Optional: observability ─────────────────────────────────────────────────
APP_ENV=development
ALLOY_URL=
ALLOY_USERNAME=
ALLOY_PASSWORD=
```

### Environment Variable Reference

| Variable               | Code Default                | Description                                      |
|------------------------|-----------------------------|--------------------------------------------------|
| `SERVER_PORT`          | `8083` (`.env` overrides to `8085`) | HTTP listen port                       |
| `DB_PATH`              | `system.db`                 | SQLite database file path                        |
| `JWT_PRIVATE_KEY_PATH` | (empty)                     | Path to RSA **private** key PEM                  |
| `JWT_PUBLIC_KEY_PATH`  | (empty)                     | Path to RSA **public** key PEM                   |
| `VALKEY_ADDR`          | `localhost:6379`            | Valkey/Redis address                             |
| `RABBITMQ_URL`         | (empty)                     | RabbitMQ connection URL                          |
| `SMTP_HOST`            | `smtp.gmail.com`            | SMTP server host                                 |
| `SMTP_PORT`            | `587`                       | SMTP server port                                 |
| `SMTP_USER`            | (empty)                     | SMTP username                                    |
| `SMTP_PASS`            | (empty)                     | SMTP password                                    |
| `EMAIL_FROM_ADDRESS`   | (empty)                     | Sender email address                             |
| `EMAIL_FROM_NAME`      | (empty)                     | Sender display name                              |
| `EMAIL_TEMPLATE_DIR`   | `templates`                 | Directory containing email HTML templates        |
| `PUBLIC_BASE_URL`      | `http://localhost:8085`     | Base URL for OpenAPI spec / Swagger UI           |
| `APP_ENV`              | `production`                | `development` / `staging` / `production`         |
| `ALLOY_URL`            | (empty)                     | Alloy Loki-receiver for log push                 |
| `ALLOY_USERNAME`       | (empty)                     | Grafana Cloud user ID                            |
| `ALLOY_PASSWORD`       | (empty)                     | Grafana Cloud API token                          |

---

## JWT Key Pair Setup

System is the **only** service that holds the private key. It signs JWTs on login; all other services (SCM, MRP, Sales) verify tokens with the public key.

```bash
# Generate a new RSA key pair (run once)
openssl genrsa -out jwt_private.pem 4096
openssl rsa -in jwt_private.pem -pubout -out jwt_public.pem
```

- Place `jwt_private.pem` where System can read it (`JWT_PRIVATE_KEY_PATH`).
- Copy `jwt_public.pem` to every other service directory and set their `JWT_PUBLIC_KEY_PATH`.
- If no private key path is configured, System generates an **ephemeral** key at startup (dev mode only -- tokens will not survive restarts).

---

## Option 1: Bare Metal

```bash
cd system/

# 1. Install Go dependencies
go mod download

# 2. Seed the database (runs migrations + seeds roles, action types, admin user)
make seed
#    Or manually:
#    go run ./cmd/migrate -db system.db
#    go run ./cmd/seeder -db system.db

# 3. Start the server
make run
#    Listening on http://localhost:8085
#    Swagger UI: http://localhost:8085/docs/
```

### Available Make Targets

| Command           | Description                                    |
|-------------------|------------------------------------------------|
| `make run`        | Start the System server                        |
| `make build`      | Compile binary to `build/zeus-system`          |
| `make seed`       | Run migrations + seed initial data             |
| `make test`       | Run all tests                                  |
| `make vet`        | Run `go vet`                                   |
| `make lint`       | Run `staticcheck`                              |
| `make clean`      | Remove build artifacts and database            |
| `make dev`        | Run with live reload via [air](https://github.com/air-verse/air) |
| `make infra`      | Start Valkey via `podman compose`              |
| `make exec-valkey`| Open `valkey-cli` in the container             |

### What the Seeder Creates

- **Roles:** `admin`, `scm_operator`, `scm_worker`, `mrp_operator`, `mrp_worker`, `sales_operator`, `sales_worker`
- **Action types:** `LOGIN`, `CREATE`, `UPDATE`, `DELETE`, etc. (for audit logging)
- **Default admin user:** created with a bcrypt-hashed password

---

## Option 2: Docker

### Build

Build from the **repo root** (the Dockerfile copies `pkg/exception` and `pkg/response` from the monorepo):

```bash
docker build -f system/Dockerfile -t zeus-system:latest .
```

### Run

```bash
# 1. Start Valkey (optional)
docker run -d --name zeus-system-valkey \
  -p 6379:6379 \
  valkey/valkey:8-alpine

# 2. Run migrations + seed
docker run --rm \
  -v zeus-system-data:/app/data \
  zeus-system:latest \
  ./seeder -db /app/data/system.db

# 3. Start the server
docker run -d --name zeus-system \
  -p 8085:8085 \
  -v zeus-system-data:/app/data \
  -e SERVER_PORT=8085 \
  -e DB_PATH=/app/data/system.db \
  -e JWT_PRIVATE_KEY_PATH=jwt_private.pem \
  -e JWT_PUBLIC_KEY_PATH=jwt_public.pem \
  -e VALKEY_ADDR=redis://zeus-system-valkey:6379 \
  -e APP_ENV=development \
  --link zeus-system-valkey \
  zeus-system:latest
```

### Docker Compose (Valkey only)

```bash
cd system/
docker compose up -d   # Valkey on port 6379
```

---

## API Documentation

- **Swagger UI:** http://localhost:8085/docs/
- **Health check:** `GET /health`

### Key Endpoints

| Endpoint | Auth | Description |
|----------|------|-------------|
| `POST /api/v1/system/auth/login` | Public | Authenticate and receive JWT |
| `POST /api/v1/system/auth/refresh` | Public | Refresh an access token |
| `POST /api/v1/system/auth/logout` | Public | Revoke a session |
| `POST /api/v1/system/auth/change-password` | JWT | Change own password |
| `POST /api/v1/system/users` | Admin | Create a new user |
| `GET /api/v1/system/users` | Admin | List all users |
| `GET /api/v1/system/logs` | Admin | Query audit logs |

---

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `failed to read private key` | Missing or invalid PEM file | Generate a key pair; set `JWT_PRIVATE_KEY_PATH` |
| `using ephemeral rsa key` warning | No private key configured | Expected in dev; provide a key file for persistence |
| `valkey connection failed` | Valkey not running | Non-fatal; sessions cached in-memory fallback |
| `rabbitmq connection failed` | RabbitMQ not running | Non-fatal; audit events not consumed from queue |
| `account email sender disabled` | SMTP not configured | Non-fatal; account creation emails will not be sent |
| CGO compilation error | Missing C toolchain | Install `build-base` + `sqlite-dev` |
