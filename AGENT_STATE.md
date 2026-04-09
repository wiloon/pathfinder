# AGENT_STATE.md

> This file is the persistent state anchor for Claude Code sessions working on the Pathfinder project.
> Update it at the start/end of each significant session or when major decisions are made.

---

## Project Context

**Pathfinder** is a goal-oriented daily planning application that uses AI to generate and adapt task lists.

### Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.24, Gin framework |
| Database | SQLite via GORM (auto-migrated) |
| Auth | gorilla/sessions (HTTPOnly cookie, 30-day TTL) |
| AI | MiniMax Chat API (OpenAI-compatible, `MiniMax-Text-01`) |
| Email | Resend API (verification + password reset) |
| Frontend | Next.js 15 (App Router), React 19, TypeScript |
| Styling | Tailwind CSS v4, shadcn/ui (Radix UI) |
| State | TanStack Query v5 (React Query) |
| Forms | react-hook-form + Zod |
| Drag & Drop | dnd-kit |
| Container | Docker / Podman, docker-compose |

### Implemented Features

- **User auth:** Register → email verify → login → password reset (Resend)
- **Goals:** Create primary/secondary goals with file attachments; set/change primary; CRUD
- **Daily plans:** AI-generated task list (4-8 tasks) on first access; regenerate on demand
- **Tasks:** Status updates (pending/done/skipped), drag-to-reorder, time slots
- **Events:** Create upcoming events with AI-generated prep tasks; retrospective flow
- **Check-ins:** Evening standup (completed/blocked/tomorrow_focus); triggers AI regeneration of next day's plan
- **User profile:** Bio + resume upload
- **Data portability:** Full export/import as JSON

### Repo Layout

```
/pathfinder
├── pathfinder-api/       # Go backend (port 8080)
│   ├── main.go           # Entry: config load, Init() all packages, route registration
│   ├── storage/          # GORM models + DB singleton
│   ├── middleware/        # Session + auth middleware
│   ├── user/             # Auth handlers + tests
│   ├── goal/             # Goal CRUD + tests
│   ├── plan/             # Daily plan handlers
│   ├── checkin/          # Check-in handlers + tests
│   ├── event/            # Event handlers
│   ├── ai/               # MiniMax API wrapper (3 functions)
│   └── email/            # Resend API wrapper
├── pathfinder-ui/        # Next.js frontend (port 3000)
│   ├── app/              # App Router pages
│   ├── components/       # Shared UI components
│   └── lib/              # Axios client, utils
├── Taskfile.yml
└── docker-compose.yml
```

---

## Evolutionary Standards

> These are non-negotiable quality constraints. ALL new code and ALL code touched during edits MUST comply.
> Violation of existing code is noted in Session Memory and fixed opportunistically.

### 1. Error Handling — No Silent Failures

```go
// WRONG
result, _ := someOperation()

// WRONG
if err != nil {
    return err  // no context
}

// CORRECT
result, err := someOperation()
if err != nil {
    return fmt.Errorf("operation context: %w", err)
}
```

- Every `error` return must be checked.
- Every propagated error must be wrapped with `fmt.Errorf("context: %w", err)` to preserve the error chain.
- Errors that terminate a request must be logged with at least one contextual field (user ID, resource ID, etc.).
- Swallowing errors (the `_` pattern) is only permitted for explicitly documented no-op cases.

### 2. Layered Architecture — No Cross-Layer Leakage

```
Handler layer  →  Service/logic layer  →  Storage layer
```

- **Handlers** (`goal.go`, `plan.go`, etc.) must only: parse input, call service functions, write HTTP response.
- **Business logic** (plan generation strategy, check-in processing) must live in dedicated functions or sub-packages, NOT inline inside handlers.
- **Database access** (`storage.DB.*`) must NOT appear directly in AI, email, or middleware packages — only in storage and the domain handler packages.
- **External API calls** (MiniMax, Resend) must NOT be inlined in handler functions; call through the `ai` / `email` packages.

### 3. Test-Driven Development

- Every new exported function that contains non-trivial logic must have a corresponding `_test.go` entry.
- Tests must use in-memory SQLite (`:memory:`) — no shared state between test cases.
- Mocking of external APIs (MiniMax, Resend) is required in tests; real network calls are forbidden in unit/integration tests.
- Table-driven tests are preferred for functions with multiple input variants.

### 4. Defensive Input Validation

- All user-supplied strings must be validated for length and format before they reach the database or AI prompt.
- File uploads must be validated: check MIME type, reject files above a size limit, sanitize filenames.
- IDs extracted from URL params must be parsed (e.g., `strconv.Atoi`) and validated before use in queries.
- Prompt injection risks: user text inserted into AI prompts must be clearly delimited (use JSON encoding, not string concatenation).

### 5. API Response Consistency

- Success responses: `{"data": ...}` or named field; always include HTTP 2xx.
- Error responses: `{"error": "human-readable message"}` with appropriate 4xx/5xx code.
- Never return raw GORM model structs (they may expose internal fields); use explicit response structs or `map[string]interface{}` with selected fields.

---

## Progress Tracking

| # | Task | Status |
|---|---|---|
| 1 | Establish and verify state anchor (create AGENT_STATE.md) | ✅ Done |
| 2 | Audit existing handlers for direct DB access violations | ⬜ Pending |
| 3 | Add input validation to user registration and goal creation | ⬜ Pending |
| 4 | Wrap AI prompt construction to prevent prompt injection | ⬜ Pending |
| 5 | Add missing test coverage for plan/ and event/ packages | ⬜ Pending |

---

## Session Memory

### Session 2026-04-09 — Initial Scan

**Understanding established:**

- The project is a functioning MVP. Core flows (auth → goal → plan → checkin loop) work end-to-end.
- The AI layer (`ai/ai.go`) is the most architecturally fragile: it constructs raw string prompts by concatenating user data, which is a prompt injection risk. It also falls back silently to default tasks on error, which obscures AI failures.
- Handler files (e.g., `goal/goal.go`, `plan/plan.go`) contain business logic inline — direct `storage.DB` calls interleaved with logic. This violates the layering standard and must be refactored incrementally.
- `checkin/checkin.go` is the most complex handler (fetches 7-day history, queries upcoming events, calls AI, upserts plan). It is a priority refactor candidate.
- Test coverage: `user/`, `goal/`, `checkin/` have `_test.go` files. `plan/` and `event/` do not. This is a gap.
- Frontend middleware correctly protects routes via session cookie presence check. However, `middleware.ts` does not verify session validity with the backend on every request — it only checks cookie existence.
- No violations were fixed this session; this was a read-only scan. Violations are catalogued above for future remediation.

**Known technical debt:**
1. `ai/ai.go` — user-provided goal text inserted directly into prompt strings (prompt injection risk).
2. `plan/plan.go`, `goal/goal.go`, `event/event.go` — DB queries inline in handler functions (layering violation).
3. `plan/`, `event/` — no test files (test coverage gap).
4. File upload handlers — no MIME type validation, no size limits enforced.
5. `checkin/checkin.go` — overly large handler function; needs decomposition.
