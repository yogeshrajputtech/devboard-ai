# DevBoard AI

A small task board — React + Go + Postgres — with an AI copilot on top,
inspired by [DevBoard](https://github.com/LondheShubham153/devboard). Same
"see how the pieces connect" spirit as the original, plus:

```
browser  →  frontend (React)  →  backend (Go API)  →  database (Postgres)
                                        │
                                        └──▶  Claude (Anthropic API)
```

## AI features

All AI calls happen **only** in the Go backend — your `ANTHROPIC_API_KEY`
never reaches the browser.

| Feature | Where | What it does |
| --- | --- | --- |
| **AI task generation** | "Generate with AI" button on a project | Describe a goal in plain English → Claude breaks it into 3-8 tasks and creates them. |
| **Smart task assistant** | ✨ menu on each task card | Improve description / suggest subtasks / summarize — always a suggestion you accept or discard, never a silent edit. |
| **Auto priority & tagging** | Same ✨ menu | Suggests a priority and 1-3 tags from the task's title & description. |
| **Natural-language search** | "AI search" toggle on a project | Ask things like *"overdue high priority tasks"* — Claude derives a filter, the backend runs it as SQL. Falls back to plain literal search if AI isn't configured. |

There's no login — same as the original DevBoard, auth is out of scope for
v1 (see [What's next](#whats-next)).

## What you need

- **Docker** (with Docker Compose).
- An **Anthropic API key** if you want the AI features live — get one at
  [console.anthropic.com](https://console.anthropic.com). Without one, the
  app still runs fully — AI endpoints just return a clear "not configured"
  response instead of crashing anything.

You do **not** need Node, Go, or Postgres installed — they all run inside
containers.

## Quickstart

```bash
cp .env.example .env
# optional: put your key in .env → ANTHROPIC_API_KEY=sk-ant-...

docker compose up --build
```

Open **http://localhost:8080**.

Or with `make` (creates `.env` for you automatically):

```bash
make up
```

| Piece    | Open in browser / curl        | Notes                                   |
| -------- | ------------------------------ | ---------------------------------------- |
| Frontend | http://localhost:8080          | the app; forwards `/api` to the backend  |
| Backend  | http://localhost:8081/health   | the Go API (the app uses it via `/api`)  |
| Postgres | localhost:5432                 | user / password: `devboard` / `devboard` |

```bash
make           # list all commands
make up        # build and start everything
make down      # stop everything
make logs      # watch the logs
make reset     # wipe the database and start fresh
make smoke     # quick check that everything (including AI wiring) works
```

Check both the backend and the frontend proxy after startup:

```bash
curl -fsS http://localhost:8081/health
curl -fsS http://localhost:8080/api/health
```

Both responses should include the same `ai_enabled` value. If only the
first request works, check `docker compose logs frontend`.

## The API

The browser calls these as `/api/...`; the backend serves them at the root.

### Core (no AI)

| Method | Path                      | What it does                          |
| ------ | ------------------------- | -------------------------------------- |
| GET    | `/projects`               | list projects                          |
| POST   | `/projects`               | create a project                       |
| GET    | `/tasks?project_id=N`     | list tasks in a project                |
| POST   | `/tasks`                  | create a task                          |
| PATCH  | `/tasks/:id`              | update a task (status, priority, tags…) |
| GET    | `/search?q=&project_id=N` | literal title/description search       |
| GET    | `/health`                 | health check (`ai_enabled` field tells you if AI is configured) |

### AI

| Method | Path                                     | Writes DB? | What it does |
| ------ | ----------------------------------------- | :--------: | ------------- |
| POST   | `/ai/generate-tasks`                      | yes        | `{project_id, prompt}` → creates 3-8 tasks |
| POST   | `/tasks/:id/ai/improve-description`       | no         | → `{suggestion}` |
| POST   | `/tasks/:id/ai/suggest-subtasks`          | no         | → `{subtasks: [...]}` |
| POST   | `/tasks/:id/ai/summarize`                 | no         | → `{summary}` |
| POST   | `/ai/suggest-priority`                    | no         | `{title, description}` → `{priority, tags}` |
| POST   | `/ai/search`                              | no         | `{project_id, query}` → `{results, filter}` |

Every AI endpoint returns `503 {"error":"ai_unavailable"}` if
`ANTHROPIC_API_KEY` isn't set, `504 {"error":"ai_timeout"}` on a slow Claude
call, and `502 {"error":"ai_upstream_error"}` / `{"error":"ai_invalid_response"}`
on other failures — none of that touches the plain CRUD endpoints above.

## Manual AI verification (needs a real API key)

These aren't run in CI (they cost tokens) — check them by hand after
`make up` with a key configured:

```bash
curl -s -X POST localhost:8081/ai/generate-tasks \
  -d '{"project_id":1,"prompt":"Prepare a Q3 launch checklist"}'

curl -s -X POST localhost:8081/ai/suggest-priority \
  -d '{"title":"Fix login bug","description":"Users cannot log in on mobile Safari"}'

curl -s -X POST localhost:8081/tasks/1/ai/improve-description -d '{}'
curl -s -X POST localhost:8081/tasks/1/ai/suggest-subtasks -d '{}'
curl -s -X POST localhost:8081/tasks/1/ai/summarize -d '{}'

curl -s -X POST localhost:8081/ai/search \
  -d '{"project_id":1,"query":"overdue high priority tasks"}'
```

Then click through the UI: the "Generate with AI" modal on a project page,
the ✨ menu on any task card, and "AI search" — confirming accept/discard
writes through to the visible board.

## Settings live in `.env`

```bash
cp .env.example .env     # or: make setup
```

`.env.example` is the template kept in git; `.env` is gitignored.

## Folder layout

```
.
├── docker-compose.yml   starts frontend + backend + postgres together
├── Makefile              short commands (make up, make down, ...)
├── .env.example           template for settings (copy to .env)
├── frontend/              React app (Vite). UI + AI/ components, forwards /api
├── backend/               Go API — main.go (CRUD) + ai.go/ai_prompts.go (Claude)
└── init/postgres/         schema (incl. tags) + example data, loaded on first start
```

## Development (without Docker)

```bash
# backend
cd backend
go mod tidy       # first time only — resolves & pins the Anthropic Go SDK
POSTGRES_URL=postgres://devboard:devboard@localhost:5432/devboard?sslmode=disable \
  ANTHROPIC_API_KEY=sk-ant-... \
  go run .

# frontend
cd frontend
npm install
npm run dev        # http://localhost:5173, proxies /api → localhost:8080
```

Run the test suites:

```bash
cd backend && go test ./...     # AI handler tests stub api.anthropic.com — no key needed
cd frontend && npm run test
```

## What's next

v1 is intentionally minimal — auth-free and AI-added, same trade-off the
original DevBoard made for "AI-free, wiring-focused." Deferred to a v2:

- **Auth / login** — none in v1.
- **Real-time sync** — no websockets; the board polls via TanStack Query's
  normal cache invalidation.
- **Richer task model** — comments, attachments, and true nested subtasks
  (`parent_task_id`) instead of AI-suggested subtasks being created as plain
  sibling tasks.
- **Normalized tags** — `tags` is a `TEXT[]` column for now; a `tags` /
  `task_tags` table would be the move if tag autocomplete or analytics
  become real features.
- **Kubernetes manifests + full DevSecOps CI** (SAST/DAST/secret/dependency
  scanning) — the original DevBoard's `k8s/` and 9-workflow suite are easy
  to port later since this keeps the same 3-service topology; v1 ships a
  single lightweight `go build/vet/test` + `npm build/test` CI job instead.
