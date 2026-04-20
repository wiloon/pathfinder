# ADR-0008: Security Task Deferrals

**Status:** Accepted
**Date:** 2026-04-20

---

## Decision

Defer the following P1 security tasks until the core feature set (tasks #16–#27) is complete:

| Violation | File | Issue |
|---|---|---|
| V1 | `ai/minimax.go` | User goal text concatenated directly into AI prompt strings (prompt injection risk) |
| V6 | `goal/goal.go`, `user/user.go` | File uploads lack MIME type allowlist check and size limit |
| V8 | `pathfinder-ui/middleware.ts` | Route protection checks cookie existence only; does not verify session validity with backend (auth bypass risk) |

## Rationale

- Pathfinder is not exposed to the internet; it runs on localhost for a single, trusted operator.
- There is no multi-user data at risk — single-tenant deployment means only the operator's own data could be affected.
- Business functionality is higher priority for the current development phase; feature completeness unlocks meaningful testing of the full system.
- Risk is negligible at current exposure level.

## Resumption Condition

When the last of tasks #16–#27 is marked Done: create a dedicated security sprint addressing V1, V6, and V8 **before any public or networked deployment**.

Priority order for remediation:
1. **V1** — Prompt injection is the highest risk if the API is ever exposed; fix by JSON-encoding user text before prompt insertion (see C4).
2. **V6** — File upload validation; add MIME allowlist and 10MB size limit.
3. **V8** — Frontend session verification; add a `/api/auth/me` check in `middleware.ts`.

## Alternatives Rejected

- Fix security first — blocks feature delivery; risk is negligible at current exposure level.
