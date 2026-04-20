# ADR-0002: Authentication Architecture

**Status:** Accepted
**Date:** Project start / 2026-04-19 / 2026-04-20

---

## Session Cookie Auth (not JWT)

**Decision:** Use `gorilla/sessions` with HTTPOnly cookies for all browser-facing authentication. Session TTL: 30 days.

**Rationale:** Simpler revocation — delete the session record. No token refresh logic needed. HTTPOnly cookie mitigates XSS. Appropriate overhead for this scale.

**Alternatives rejected:**
- JWT — stateless but harder to revoke; overkill for this scale.

---

## ServiceTokenAuth as a Separate Middleware

**Decision:** Machine-to-machine auth (OpenClaw, agent scripts) uses a separate `ServiceTokenAuth` middleware, not an OR condition merged into `RequireAuth`.

Route groups:
- Browser routes → `RequireAuth` only
- OpenClaw/agent routes → `ServiceTokenAuth` only
- Routes open to both → mount both middlewares

**Rationale:** Keeps auth paths explicit and auditable. Token leakage cannot grant access to routes not explicitly opted in. Easier to reason about which routes accept machine auth.

**Alternatives rejected:**
- OR condition inside `RequireAuth` — token leakage grants access to all user routes; harder to audit.

---

## Single-Tenant Deployment

**Decision:** One Pathfinder instance = one OpenClaw backend = one user. `config.toml` has a single `service_user_id`. No per-user token routing.

**Rationale:** Project is single-user. Service token model stays trivially simple. No multi-user data isolation is needed.

**Alternatives rejected:**
- Per-user token registry — overkill; no other users exist.
