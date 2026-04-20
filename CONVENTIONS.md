# CONVENTIONS.md

> **Harness Engineering — Coding Conventions**
> Read this file before writing any code in this repository.
> These are the **target standards** for all new code and all code touched during edits.
> They describe where the codebase is going, not where it is today.
> For confirmed deviations from these standards, see **Known Violations** in `AGENT_STATE.md`.

---

## C1 — No Silent Error Swallowing

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

---

## C2 — Handler / Logic / Storage Separation

```
HTTP Handler → business logic function → storage.DB calls
```

- Handlers parse input, call logic functions, write HTTP response. Nothing else.
- Business logic lives in named functions (not inlined in handlers).
- `storage.DB.*` must not appear in `ai/` or `email/` packages.
- External API calls (MiniMax, Resend) must go through `ai`/`email` packages, not inline in handlers.

---

## C3 — Tests Required for Non-Trivial Logic

- New exported functions with non-trivial logic → `_test.go` entry required.
- Tests use in-memory SQLite (`:memory:`); no shared state between cases.
- External APIs (MiniMax, Resend) must be mocked; real network calls forbidden in tests.
- Prefer table-driven tests for multi-variant inputs.

---

## C4 — Defensive Input Validation

- All user-supplied strings: validate length and format before DB or AI prompt.
- File uploads: validate MIME type, enforce size limit, sanitize filename.
- URL param IDs: always parse with `strconv.Atoi` and validate before querying.
- AI prompts: user text must be JSON-encoded before insertion — never concatenated as raw strings.

---

## C5 — Consistent API Responses

- Success: named JSON field + HTTP 2xx.
- Error: `{"error": "human-readable message"}` + appropriate 4xx/5xx.
- Never return raw GORM model structs (may expose internal fields).
