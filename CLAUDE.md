# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

All commands use [Task](https://taskfile.dev) (`task`). Run `task` (no args) to list available tasks.

**Backend (Go):**
```sh
task api                  # Start API server (port 8080)
task test                 # Run backend tests with verbose output
task test:short           # Run backend tests without verbose output
task test:cover           # Run backend tests with coverage report
# Run a single test package:
cd pathfinder-api && go test ./goal/ -v -run TestGoalName -count=1
```

**Frontend (Next.js):**
```sh
task ui                   # Install deps and start dev server (port 3000)
cd pathfinder-ui && pnpm lint        # Lint
cd pathfinder-ui && pnpm build       # Production build
task test:e2e:install     # Install Playwright browsers (run once)
task test:e2e             # Run Playwright E2E tests
cd pathfinder-ui && pnpm test:e2e:ui # Playwright interactive UI mode
```

**Full stack:**
```sh
task docker-compose up    # Run both services in containers
```

## Configuration

The API reads `pathfinder-api/config.toml` by default. Override with `CONFIG_PATH` env var. Config sections: `[server]`, `[database]`, `[ai]`, `[resend]`, `[app]`.

The AI integration targets MiniMax API (OpenAI-compatible). Configure `base_url`, `api_key`, and `model` under `[ai]`.

## Architecture

Monorepo with two independent applications:

### `pathfinder-api/` — Go REST API

- **Entry point:** `main.go` — loads config, inits all packages, registers routes
- **Route structure:** All authenticated routes are grouped under `/api` with `middleware.RequireAuth()`. Auth routes (`/api/auth/*`) are public.
- **Package layout:** Each domain package (`user`, `goal`, `plan`, `event`, `checkin`, `ai`, `email`) is initialized with `Init(...)` in `main.go` and receives dependencies at startup.
- **Storage:** Single `storage` package owns all GORM models (`models.go`) and the global `storage.DB`. All other packages import `storage` for data access — there is no repository abstraction layer.
- **Auth:** Session-based via `gorilla/sessions` (httpOnly cookie). `middleware.RequireAuth()` validates session and sets `user_id` on the Gin context. Handlers retrieve it with `c.GetString("user_id")`.
- **AI:** `ai/ai.go` wraps MiniMax calls. Used by `plan` and `checkin` packages to generate daily tasks and replan based on check-in input.

### `pathfinder-ui/` — Next.js Frontend

- **Framework:** Next.js 15 App Router, React 19, TypeScript
- **Styling:** Tailwind CSS v4, shadcn/ui (Radix UI primitives)
- **Data fetching:** TanStack Query (React Query) for server state; axios for HTTP
- **Forms:** react-hook-form + Zod validation
- **Drag-and-drop:** dnd-kit (used for task reordering in daily plan)
- **Notifications:** sonner (toast)
- **E2E tests:** Playwright under `pathfinder-ui/e2e/`

### Data model summary

| Model | Key fields |
|---|---|
| `User` | status: `pending\|active`; email verification flow |
| `Goal` | type: `primary\|secondary`; status: `active\|paused\|completed` |
| `DailyPlan` | date (YYYY-MM-DD); has many `Task` |
| `Task` | status: `pending\|done\|skipped`; linked to a `Goal` |
| `Event` | status: `upcoming\|completed`; supports retro notes |
| `CheckIn` | daily standup: completed, blocked, tomorrow_focus |
