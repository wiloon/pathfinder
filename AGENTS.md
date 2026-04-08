# AI Agent Instructions for Pathfinder

## Project Overview

Pathfinder is an AI-powered personal goal and productivity tracker consisting of:

- **pathfinder-api/** — Go backend (REST API, goal/plan/checkin management, AI plan generation)
- **pathfinder-ui/** — Next.js 15 frontend (TypeScript, Tailwind CSS, shadcn/ui)

### Technology Stack

**Backend:**
- Go 1.24
- Gin v1.10 (HTTP framework)
- GORM + glebarez/sqlite (SQLite, pure Go driver)
- gorilla/sessions (cookie-based auth)
- bcrypt (password hashing)
- Resend API (transactional email)
- MiniMax API (AI plan generation)

**Frontend:**
- Next.js 15 with Turbopack
- React 19, TypeScript
- Tailwind CSS + shadcn/ui components
- @tanstack/react-query (server state)
- react-hook-form + zod (form validation)
- @dnd-kit (drag-and-drop task ordering)
- Playwright (E2E tests)

## Repository Structure

```
pathfinder/
├── pathfinder-api/        # Go backend
│   ├── ai/                # AI plan generation (MiniMax)
│   ├── checkin/           # Daily check-in handlers
│   ├── email/             # Email sending via Resend
│   ├── event/             # Events and retros
│   ├── goal/              # Goal CRUD handlers
│   ├── middleware/        # Logger, Session, RequireAuth
│   ├── plan/              # Daily plan generation and task management
│   ├── storage/           # GORM models and DB init
│   ├── user/              # Auth: register, login, verify email, reset password
│   ├── config.toml        # Local config (not committed)
│   ├── config.example.toml
│   └── main.go            # Route registration, server startup
├── pathfinder-ui/         # Next.js frontend
│   ├── app/               # Next.js App Router pages
│   │   ├── login/
│   │   ├── register/
│   │   ├── onboarding/
│   │   ├── today/         # Today's plan page
│   │   ├── goals/
│   │   ├── events/
│   │   ├── checkin/
│   │   └── verify-email/
│   ├── components/
│   │   ├── ui/            # shadcn/ui primitives
│   │   ├── navbar.tsx     # Auth-aware navigation bar
│   │   └── add-goal-dialog.tsx  # Reusable add-goal dialog
│   ├── lib/
│   │   ├── api.ts         # Axios API client and all API functions
│   │   └── utils.ts
│   ├── middleware.ts       # Next.js route protection (redirects to /login)
│   └── e2e/               # Playwright E2E tests
└── Taskfile.yml           # Task runner (see Build Commands)
```

## Build and Run Commands

```bash
# Start backend API (port 8080)
task api

# Start frontend dev server (port 3000)
task ui

# Run all Go unit/integration tests
task test

# Run tests (short output)
task test:short

# Run tests with coverage
task test:cover

# Install Playwright browsers (run once)
task test:e2e:install

# Run E2E tests
task test:e2e
```

Manual commands:
```bash
# Backend
cd pathfinder-api && go build ./...
cd pathfinder-api && go test ./user/ ./goal/ ./checkin/ -v -count=1

# Frontend
cd pathfinder-ui && pnpm install && pnpm dev
cd pathfinder-ui && pnpm tsc --noEmit
```

## Authentication Architecture

- **Session-based**: gorilla/sessions stores `user_id` in a signed cookie (`pathfinder-session`)
- **RequireAuth middleware**: all `/api/*` routes except auth routes require a valid session
- User status: `pending` (email not verified) → `active`
- Email verification via Resend API; token expires in 48h
- Password reset token expires in 1h

### Protected vs Public Routes

```
Public  → POST /api/auth/register, /login, /logout
          GET  /api/auth/me, /api/auth/verify-email
          POST /api/auth/resend-verification, /forgot-password, /reset-password

Protected (RequireAuth) → /api/goals/*, /api/plan/*, /api/tasks/*,
                          /api/events/*, /api/checkin/*, /api/user/profile,
                          /api/export, /api/import
```

### Frontend Route Protection

`middleware.ts` at the root of `pathfinder-ui` intercepts all non-public routes. If `pathfinder-session` cookie is absent, redirects to `/login?from=<original-path>`.

## Key Patterns

### Reading user_id in handlers

All protected handlers obtain the user via gin context:
```go
userID := c.GetString("user_id")
```
Never use hardcoded `"local"` or read from the session directly in handler files.

### API client (frontend)

All API calls go through `lib/api.ts` using the shared axios instance with `withCredentials: true`. Add new endpoints there, not inline in components.

### Auth state in React

Navbar and any component that needs auth state uses:
```typescript
const { data: me } = useQuery({ queryKey: ['me'], queryFn: authGetMe, retry: false });
```
After login/register/logout, call `queryClient.invalidateQueries({ queryKey: ['me'] })`.

### Reusable components

- `<AddGoalDialog trigger={...} />` — opens add-goal form in a dialog; accepts optional `trigger` prop and `onSuccess` callback

## Configuration

`pathfinder-api/config.toml` (local only, gitignored):
```toml
[server]
port = "8080"
session_secret = "change-in-production"

[database]
dsn = "pathfinder.db"   # use ":memory:" in tests

[ai]
api_key = "..."
model = "MiniMax-Text-01"
base_url = "https://api.minimaxi.chat/v1"

[resend]
api_key = ""
from = "Pathfinder <noreply@yourdomain.com>"

[app]
frontend_base_url = "http://localhost:3000"
```

Use `config.example.toml` as the committed template; never commit real secrets.

## Testing

### Go tests
- Test files: `user/user_test.go`, `goal/goal_test.go`, `checkin/checkin_test.go`
- Use `storage.Init(":memory:")` for in-memory SQLite; no mocking needed
- `TestMain` in `user/user_test.go` initializes session store
- Pattern: white-box tests in `package user`; black-box in `package goal_test` / `package checkin_test`

### E2E tests
- Config: `pathfinder-ui/playwright.config.ts`
- Test files: `e2e/register.spec.ts`, `e2e/login.spec.ts`, `e2e/onboarding.spec.ts`
- Runs against `pnpm dev` via `webServer` config
- Use `.first()` when a locator could match multiple elements (strict mode)

## Security Notes

- Session secret must be changed in production
- All user-supplied data is parameterized (GORM handles this)
- Passwords hashed with bcrypt (cost 12)
- `HttpOnly` cookies, `SameSite: Lax`
- CORS restricted to `http://localhost:3000` in development

## AI Assistant Guidelines

1. All code, comments, and commit messages must be in **English**
2. Never hardcode `userID = "local"` — always use `c.GetString("user_id")`
3. New API endpoints go in `main.go` under the `api` group (protected) or the auth block (public)
4. New frontend API functions go in `lib/api.ts`
5. After login/register/logout, invalidate `['me']` query so Navbar updates
6. Run `go build ./...` and `pnpm tsc --noEmit` after changes to verify no errors
