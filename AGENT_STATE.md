# AGENT_STATE.md

> **Harness Engineering State Anchor** — Read this file at the start of every session before touching any code.
> Update "Current Focus" and "Session Memory" when starting or finishing significant work.

---

<!-- CONTEXT BUDGET: Section reading priority when context is tight -->
<!-- MUST READ: Current Focus, Known Violations, Progress Tracking -->
<!-- READ IF RELEVANT: Conventions, Decision Log -->
<!-- SKIP IF PRESSED: Session Memory older than 2 sessions -->

---

## Current Focus

**Active task:** None — awaiting instruction.  
**Last completed:** Recorded brief planning decisions (sync HTTP, DB storage, structured response) + added tasks #15–#17 (2026-04-19).  
**Recommended next task:** `#9` — Add service token middleware (unblocks #10, #11, #13; enables OpenClaw to call Pathfinder on user's behalf, and agent/curl access without browser session).

**Blockers:** None.

---

## Project Context

**Pathfinder** is a goal-oriented daily planning application. The core loop: user sets goals → AI generates a daily task plan → user completes tasks → evening check-in triggers AI to replan for tomorrow.

### Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.24, Gin framework |
| Database | SQLite via GORM (auto-migrated, no migrations files) |
| Auth | gorilla/sessions (HTTPOnly cookie, 30-day TTL) |
| AI | MiniMax Chat API (OpenAI-compatible) |
| Email | Resend API (verification + password reset) |
| Frontend | Next.js 15 App Router, React 19, TypeScript |
| Styling | Tailwind CSS v4, shadcn/ui (Radix UI primitives) |
| State | TanStack Query v5 |
| Forms | react-hook-form + Zod |
| Drag & Drop | dnd-kit (task reordering) |
| Container | Docker / Podman, docker-compose |

### Implemented Features

- **User auth:** Register → email verify → login → password reset
- **Goals:** Create `primary`/`secondary` goals with file attachments; CRUD; set primary
- **Daily plans:** AI-generated task list on first access; regenerate on demand; drag-to-reorder
- **Tasks:** Status `pending`/`done`/`skipped`; suggested time slots
- **Events:** Create upcoming milestones; retrospective notes after completion
- **Check-ins:** Evening standup (completed / blocked / tomorrow_focus) → triggers AI replanning for next day
- **User profile:** Bio + resume upload
- **Data portability:** Full JSON export / import

### Repo Layout

```
/pathfinder
├── pathfinder-api/       # Go backend (port 8080)
│   ├── main.go           # Config load → Init() all packages → register routes
│   ├── storage/          # GORM models (models.go) + DB singleton (storage.go)
│   ├── middleware/        # Session init, Logger, RequireAuth
│   ├── user/             # Auth handlers + tests
│   ├── goal/             # Goal CRUD + tests
│   ├── plan/             # Daily plan handlers (NO tests yet)
│   ├── checkin/          # Check-in handlers + tests
│   ├── event/            # Event handlers (NO tests yet)
│   ├── ai/               # MiniMax API wrapper: ChatCompletion, GenerateInitialPlan, RegenerateAfterCheckin
│   └── email/            # Resend API wrapper
├── pathfinder-ui/        # Next.js frontend (port 3000)
│   ├── app/              # App Router pages (today, goals, checkin, events, …)
│   ├── components/       # Shared UI (add-goal-dialog, navbar, shadcn/ui)
│   └── lib/              # axios client (api.ts), utils
├── Taskfile.yml          # All dev commands — use `task` not raw go/pnpm
└── docker-compose.yml
```

---

## Conventions

> These are the **target standards** for all new code and all code touched during edits.  
> They describe where the codebase is going, not where it is today.  
> See **Known Violations** below for the current gap.

### C1 — No Silent Error Swallowing

```go
// FORBIDDEN
result, _ := someOperation()

// FORBIDDEN — no context
if err != nil { return err }

// REQUIRED
result, err := someOperation()
if err != nil {
    return fmt.Errorf("createGoal: %w", err)
}
```

Every error must be checked. Propagated errors must be wrapped with `fmt.Errorf("context: %w", err)`. Errors ending a request must log at least one context field (userID, resourceID).

### C2 — Handler / Logic / Storage Separation

```
HTTP Handler → business logic function → storage.DB calls
```

- Handlers parse input, call logic functions, write HTTP response. Nothing else.
- Business logic lives in named functions (not inlined in handlers).
- `storage.DB.*` must not appear in `ai/` or `email/` packages.
- External API calls (MiniMax, Resend) must go through `ai`/`email` packages, not inline in handlers.

### C3 — Tests Required for Non-Trivial Logic

- New exported functions with non-trivial logic → `_test.go` entry required.
- Tests use in-memory SQLite (`:memory:`); no shared state between cases.
- External APIs (MiniMax, Resend) must be mocked; real network calls forbidden in tests.
- Prefer table-driven tests for multi-variant inputs.

### C4 — Defensive Input Validation

- All user-supplied strings: validate length and format before DB or AI prompt.
- File uploads: validate MIME type, enforce size limit, sanitize filename.
- URL param IDs: always parse with `strconv.Atoi` and validate before querying.
- AI prompts: user text must be JSON-encoded before insertion — never concatenated as raw strings.

### C5 — Consistent API Responses

- Success: named JSON field + HTTP 2xx.
- Error: `{"error": "human-readable message"}` + appropriate 4xx/5xx.
- Never return raw GORM model structs (may expose internal fields).

---

## Known Violations

> These are **confirmed deviations** from the conventions above that exist in the current codebase.  
> Do not treat them as acceptable patterns. Fix them when touching the relevant file, or as dedicated tasks.

| ID | File | Violation | Convention | Priority |
|---|---|---|---|---|
| V1 | `ai/ai.go` | User goal text concatenated directly into prompt strings | C4 (prompt injection risk) | P1 — Security |
| V2 | `plan/plan.go` | Multiple `storage.DB.*` calls inline inside handler functions | C2 (layering) | P2 |
| V3 | `goal/goal.go` | Business logic (plan trigger on first goal) inline in handler | C2 (layering) | P2 |
| V4 | `event/event.go` | No test file exists | C3 (test coverage) | P2 |
| V5 | `plan/plan.go` | No test file exists | C3 (test coverage) | P2 |
| V6 | `goal/goal.go`, `user/user.go` | File uploads lack MIME type check and size limit | C4 (input validation) | P1 — Security |
| V7 | `checkin/checkin.go` | Single handler function fetches history, queries events, calls AI, upserts plan — too large | C2 (logic separation) | P3 |
| V8 | `pathfinder-ui/middleware.ts` | Route protection checks cookie existence only; does not verify session with backend | C4 (auth bypass risk) | P2 |

---

## Decision Log

> Key architectural decisions and their rationale. Do not reverse these without discussion.

| Date | Decision | Rationale | Alternatives rejected |
|---|---|---|---|
| Project start | SQLite as database | Zero-ops for MVP; single-user or small-team usage; easy file-based backup | Postgres (operational overhead not justified at this scale) |
| Project start | No repository abstraction layer | Reduce boilerplate for small codebase; direct `storage.DB` access is acceptable at MVP scale | Repository pattern (adds indirection without current benefit; can be introduced incrementally if packages grow) |
| Project start | Session cookie auth (not JWT) | Simpler revocation; no token refresh logic needed; HTTPOnly cookie mitigates XSS | JWT (stateless but harder to revoke; overkill for this scale) |
| Project start | MiniMax API (OpenAI-compatible) | Client requirement; interface is drop-in compatible with OpenAI SDK patterns | OpenAI directly (different provider; same interface) |
| Project start | Monorepo (api + ui in one repo) | Simplifies local dev and CI; single `Taskfile.yml` orchestrates both | Separate repos (unnecessary coordination cost at this team size) |
| 2026-04-19 | AI backend is a configurable provider, not hardcoded | Pathfinder is a UI + data layer; the decision-making brain is pluggable. User can choose MiniMax (built-in) or OpenClaw (self-hosted) as the AI backend. Both must implement the same interface in `ai/`. | Hardcode MiniMax only (blocks OpenClaw integration); Make OpenClaw the only option (breaks standalone usage) |
| 2026-04-19 | OpenClaw interacts with Pathfinder via service token, not session cookie | **Primary use case:** user talks to OpenClaw directly (no Pathfinder UI open); OpenClaw calls Pathfinder's API to create/update goals and tasks on the user's behalf. Session cookies are user-facing browser auth and cannot be used by a server-side caller. Service token is the simplest machine-to-machine credential: a single config value, one middleware check, maps to a fixed `service_user_id`. **Secondary use case:** agent/curl access for manual testing and scripting. | OAuth2 client credentials (over-engineered for single-tenant use); reuse session cookie (not suitable for server-side callers) |
| 2026-04-19 | Pathfinder → OpenClaw uses two patterns: webhook for notifications, sync HTTP for brief planning | Webhook (fire-and-forget) suits event notifications (goal.created, checkin.submitted) where result is not needed immediately. Sync HTTP suits brief planning where UI must show result to user — spinner is better UX than polling. See `pathfinder-api/docs/openclaw-integration.md`. | Pure async for all calls (requires polling, worse UX for brief flow); pure sync for all calls (blocks on notifications unnecessarily) |
| 2026-04-19 | Brief text is stored in DB | Brief history gives OpenClaw context across sessions ("last week job hunting, this week interview prep"). Without storage, each call is stateless and OpenClaw cannot detect context drift. | Fire-and-forget, no storage (simpler but loses context history) |
| 2026-04-19 | Brief → OpenClaw returns structured JSON `{goals, tasks}` parsed by Pathfinder | Pathfinder owns the data model; OpenClaw must not write directly to DB. Sync response is the cleanest contract: one request, one response, Pathfinder stores everything. | OpenClaw calls back via service token (async, needs polling, higher complexity for brief flow) |
| 2026-04-19 | Brief planning provider is a config choice (openclaw/minimax), not a fallback chain; MiniMax brief deferred | The two providers have different interfaces and capabilities. Brief (free-text → goals+tasks) requires OpenClaw. Implementing a parallel MiniMax brief path now adds complexity with no immediate user value. 501 is the correct response for the unimplemented path — explicit, not silent. | Silent fallback to MiniMax (hides misconfiguration); implement both now (unnecessary scope) |
| 2026-04-19 | Service token auth uses a separate `ServiceTokenAuth` middleware, not merged into `RequireAuth` | Keeps auth paths explicit: browser routes mount `RequireAuth`, OpenClaw/agent routes mount `ServiceTokenAuth`, routes open to both mount both. Token leakage cannot access routes not explicitly opted in. Easier to audit which routes accept machine auth. | OR condition inside `RequireAuth` (token leakage grants access to all user routes; harder to audit) |

---

## Progress Tracking

> Priority: P1 = security/correctness blocker · P2 = quality debt · P3 = nice-to-have  
> Dependencies listed where a task cannot start until another finishes.

| # | Task | Priority | Depends on | Status |
|---|---|---|---|---|
| 1 | Establish and verify state anchor | — | — | ✅ Done |
| 2 | Audit all handlers for direct DB access violations (document V2, V3, V7) | P2 | — | ✅ Done (see Known Violations) |
| 3 | Add input length/format validation to user registration and goal creation | P1 | — | ⬜ Pending |
| 4 | Fix file upload handlers: enforce MIME allowlist and size limit (V6) | P1 | — | ⬜ Pending |
| 5 | Wrap AI prompt construction to use JSON encoding, not string concat (V1) | P1 | — | ⬜ Pending |
| 6 | Add test coverage for `plan/` package | P2 | — | ⬜ Pending |
| 7 | Add test coverage for `event/` package | P2 | — | ⬜ Pending |
| 8 | Decompose `checkin/checkin.go` `SubmitCheckin` into smaller functions (V7) | P3 | #6 | ⬜ Pending |
| 9 | Add service token middleware — machine-to-machine auth for OpenClaw/agent access | P1 | — | ⬜ Pending |
| 10 | Add `POST /api/plan/tasks` — manually create a task in today's plan (no AI) | P2 | #9 | ⬜ Pending |
| 11 | Add `PATCH /api/tasks/:id` fields: title, time_slot (current PUT only updates status) | P2 | #9 | ⬜ Pending |
| 12 | Refactor `ai/` package to support pluggable provider (MiniMax or OpenClaw) via config | P2 | — | ⬜ Pending |
| 13 | Add OpenClaw webhook dispatcher — when `openclaw.webhook_url` is configured, POST goal/plan events to OpenClaw instead of calling MiniMax directly | P2 | #9 | ⬜ Pending |
| 14 | Add `[openclaw]` config section (`webhook_url`, `webhook_secret`, `sync_url`, `service_token`, `service_user_id`) and `app.ai_provider` field | P2 | — | ⬜ Pending |
| 15 | Add `PlanBrief` model to storage — fields: id, user_id, text, created_at | P2 | — | ⬜ Pending |
| 16 | Add `POST /api/plan/brief` — save brief to DB, call OpenClaw sync (`sync_url`), parse `{goals, tasks}` response, create goals + tasks; return 501 if `ai_provider` is not `openclaw` | P2 | #15 | ⬜ Pending |
| 17 | Add brief input UI on Today page — textarea + submit button, loading spinner while waiting, renders new goals/tasks on response | P2 | #16 | ⬜ Pending |

---

## Session Memory

### Session 2026-04-19 — Harness Engineering rewrite

- Rewrote AGENT_STATE.md to full Harness Engineering standard.
- Added: Current Focus, Known Violations table, Decision Log, priority + dependency columns in Progress Tracking, context budget hints.
- Merged audit findings from Session 2026-04-09 into Known Violations table (V1–V8); marked task #2 Done.
- No code was modified this session.

---

### Session 2026-04-18 — State anchor re-verification

- Re-scanned full codebase. Architecture and data model unchanged from 2026-04-09.
- No new features added; five pending tasks remained open at end of session.

---

### Session 2026-04-09 — Initial scan

- Project is a functioning MVP. Core flow (auth → goal → plan → checkin loop) works end-to-end.
- `ai/ai.go`: constructs prompts via string concatenation (prompt injection risk); silently falls back to default tasks on AI error.
- Handler files contain business logic and `storage.DB` calls inline — layering violation throughout.
- `checkin/checkin.go` is the most complex handler and highest-priority refactor candidate.
- Test coverage: `user/`, `goal/`, `checkin/` have tests. `plan/`, `event/` do not.
- `middleware.ts` (frontend) only checks cookie presence, not backend session validity.
- Session was read-only; no code modified.
