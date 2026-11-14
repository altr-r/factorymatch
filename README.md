## Tech stack (from the requirements doc)

| Layer | Choice |
| ----- | ------ |
| Backend language | Go 1.25.1 |
| Frontend framework | React + Vite |
| Styling | Tailwind CSS v4 |
| Database | PostgreSQL 16 |
| ORM | GORM |
| Deployment | Railway / AWS |

## Layout

```
factorymatch/
  docker-compose.yml   Postgres for local dev
  backend/             Go API (net/http + GORM) — see backend/README.md
    cmd/server/        entry point
    internal/          config, db, models, auth, middleware, handlers
  frontend/            React + Vite + Tailwind — see frontend/README.md
    src/
      pages/           one page per feature (C.1/C.2, Modules 1–4)
      components/      Layout, Navbar, Sidebar, ProtectedRoute
      services/api.js  every backend endpoint
      context/         AuthContext (login state)
```

## Run it

```bash
# 1. Database
docker compose up -d db

# 2. Backend (terminal 1)
cd backend
cp .env
go run ./cmd/server        # :8080, /healthz

# 3. Frontend (terminal 2)
cd frontend
cp .env.example .env
npm install
npm run dev                # http://localhost:5173
```



