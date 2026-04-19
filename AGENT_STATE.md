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
**Last completed:** Rewrote AGENT_STATE.md to Harness Engineering standard (2026-04-19).  
**Recommended next task:** `#3` — Add input validation to user registration and goal creation (highest security impact, self-contained).

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
