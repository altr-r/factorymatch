# FactoryMatch — backend (Go)

REST API for the FactoryMatch platform (CSE471 Group 8). Go 1.25.1, standard
library `net/http` for routing, GORM + PostgreSQL for storage.

**Status: structure only.** The folders, files and route lists are in place;
the implementations are TODOs to code along with. When a feature is wanted
in full, that is a separate ask.

## Run

```bash
docker compose up -d db   # from the repo root
cp .env.example .env
go run ./cmd/server        # :8080, /healthz
```

Without `DATABASE_URL` the server still starts and answers `/healthz`;
DB-backed routes return 503 until the database is configured.

## Layout

```
cmd/server/            entry point
internal/config/       env config (PORT, DATABASE_URL, JWT_SECRET, FRONTEND_ORIGIN)
internal/db/           GORM open + AutoMigrate
internal/models/       entities for C.1/C.2 and Modules 1–4
internal/auth/         bcrypt passwords, HMAC-SHA256 JWT issue/verify
internal/middleware/   logging.go, cors.go, auth.go — one file per concern
internal/services/     domain rules (DDD-lite): one function per rule, so
                       handlers stay thin and rules are testable without HTTP
internal/handlers/     routes + handlers, one file per feature area
```

## Notes

- Endpoints use the `/api/...` paths agreed on the Jira board (FAC cards).
- `go.mod` is pinned to Go 1.25.1: the latest x/crypto needs Go 1.26, so
  do not run `go get -u` on x/crypto / x/text / x/sync.
- Uploaded photos live in `./uploads/` (gitignored), served at
  `/uploads/<file>`. Production needs real private file storage.
- JWTs are stateless: logout just tells the client to drop the token.
