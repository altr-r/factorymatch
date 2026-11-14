# FactoryMatch — Smart Manufacturing Sourcing & Production Management Platform
# CSE471 Group 8, Lab Section 5, Fall 2026

B2B platform connecting buyers seeking custom-manufactured components with
verified local manufacturers and fabrication workshops in Bangladesh.

## Requirements document

`../user/files/Group8_FactoryMatch_CSE471_Functional_Requirements.docx-2.pdf`
(also kept as `Group8_FactoryMatch_CSE471_Functional_Requirements.docx.pdf`).

**Checked 2026-11-14: the new file is identical to the old one.** Same 5 pages,
same producer, extracted text byte-identical (md5 match, layout and raw
modes). Only the file size differs (209,119 vs 205,662 bytes) — a re-export
artifact, not a content change. No requirement was added, removed or changed.

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
cp .env.example .env
go run ./cmd/server        # :8080, /healthz

# 3. Frontend (terminal 2)
cd frontend
cp .env.example .env
npm install
npm run dev                # http://localhost:5173
```



