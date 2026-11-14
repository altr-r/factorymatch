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


# Contributing to FactoryMatch

## Who owns what

| Area | Owner | Jira |
| ---- | ----- | ---- |
| C.1 Registration & Login | Arafat Rahman Siam | FAC-8 |
| 1.2 Workshop Profile | Arafat Rahman Siam | FAC-6 |
| 1.1 Buyer Profile | Abdullah Al Noman | FAC-5 |
| 1.4 Workshop Search | Abdullah Al Noman | FAC-3 |
| 1.3 Capability Catalogue | Ziaur Rahman Zia | FAC-7 |
| 1.5 Shortlist | Abrar Md. Mohtasim Mahir | FAC-4 |

Work only on your own area. If a change touches someone else's feature, talk to them first.

## Branch workflow

Never push directly to `main`. (If the default branch is named something else, use that instead of `main` below.)

```bash
git checkout main
git pull
git checkout -b feature/FAC-15-registration-api
```

Branch names:

| Kind | Format | Example |
| ---- | ------ | ------- |
| Feature | `feature/<Jira-key>-<short-name>` | `feature/FAC-15-registration-api` |
| Fix | `fix/<Jira-key>-<short-name>` | `fix/FAC-18-login-401-message` |
| Docs | `docs/<short-name>` | `docs/readme-backend-setup` |

One branch per feature or fix. A branch named after the Jira card makes it obvious what it is for.

## Before you push

```bash
gofmt -l backend/          # must print nothing
cd backend && go vet ./... && go build ./...
cd frontend && npm run build
```

## Commit and open a PR

```bash
git add <only the files you changed>
git commit -m "Add registration endpoint (FAC-15)"
git push -u origin feature/FAC-15-registration-api
```

Then open a pull request to `main` on GitHub. In the PR description, write:

- What the change does
- The Jira card (e.g. FAC-15)
- How you tested it (Postman collection, commands run)

Ask one teammate to review before merging. After merge, delete the branch:

```bash
git checkout main
git pull
git branch -d feature/FAC-15-registration-api
```

## Rules

- Never commit `.env`, `node_modules/`, `dist/`, or `uploads/` — they are already in `.gitignore`.
- Keep PRs small: one feature, one review.
- Don't rewrite history on `main` (no force-push).
- If a step in `PLAN-C1-1.2.md` turns out wrong, fix the plan in the same PR.


