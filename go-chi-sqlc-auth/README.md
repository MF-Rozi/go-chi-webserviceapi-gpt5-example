# go-chi-sqlc-auth

Web service API generated from one prompt. Uses Chi, PGX, SQLC (queries included), bcrypt, JWT auth, RBAC, godotenv, and request samples.

## Stack

- Go 1.22
- Chi router
- PGX and pgxpool
- SQLC for query-to-code (queries + schema included)
- JWT auth with RBAC
- bcrypt password hashing
- godotenv for .env loading

## Setup

1. Copy `.env.example` to `.env` and edit values.
2. Ensure PostgreSQL is running and a database exists (DB_NAME in `.env`).
3. Run migrations (manually via your tool of choice, or using psql): apply files in `db/migrations` in order.
4. Install deps and run:

```powershell
# Windows PowerShell
cd go-chi-sqlc-auth
go mod tidy
go run .
```

Server listens on `PORT` (default 8080).

## Endpoints

- `GET /health` – health check
- `POST /auth/register` – create account, returns JWT and role
- `POST /auth/login` – returns JWT and role
- `GET /auth/me` – current user (JWT)
- `/users` – CRUD; list/delete are admin-only

Admin and demo users are seeded at startup if missing:

- admin: `admin@example.com` / `AdminPass123!`
- demo: `demo@example.com` / `DemoPass123!`

## Testing

Open `requests.http` in VS Code (REST Client) or use Postman/Insomnia. The file has named login and token interpolation.

## SQLC

Run `sqlc generate` if you wish to generate code. This project currently uses plain pgx for brevity, but includes `sqlc.yaml` and queries for future generation.

## SSL

Set `DB_SSLMODE` and optional `DB_SSLROOTCERT`, `DB_SSLCERT`, `DB_SSLKEY` to enable verify modes.
