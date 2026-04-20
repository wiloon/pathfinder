# ADR-0004: OpenClaw Integration Patterns

**Status:** Accepted
**Date:** 2026-04-19

> See also: `pathfinder-api/docs/openclaw-integration.md` for protocol diagrams and config reference.

---

## OpenClaw Uses Service Token, Not Session Cookie

**Decision:** OpenClaw authenticates to Pathfinder's API using a static service token passed as `X-Service-Token` header, not a browser session cookie.

**Rationale:** Primary use case: user talks to OpenClaw directly (no Pathfinder UI open); OpenClaw calls Pathfinder's API on the user's behalf. Session cookies are browser-facing and cannot be used by a server-side caller. Service token is the simplest machine-to-machine credential: a single config value, one middleware check, maps to a fixed `service_user_id`.

**Alternatives rejected:**
- OAuth2 client credentials — over-engineered for single-tenant use.
- Reuse session cookie — not suitable for server-side callers.

---

## Two Communication Patterns: Webhook + Sync HTTP

**Decision:**

| Pattern | Used for | Delivery semantics |
|---|---|---|
| **Webhook (delivery-confirmed)** | Goal created, check-in submitted | Pathfinder POSTs to `webhook_url`, waits for HTTP 2xx; failure propagates to user's operation |
| **Synchronous HTTP** | Brief planning | Pathfinder POSTs to `sync_url`, waits for full structured response (30s timeout) |

OpenClaw acknowledges webhook receipt immediately and processes AI logic asynchronously. Pathfinder never waits for AI output from a webhook.

**Rationale:** Webhooks with delivery confirmation suit event notifications: Pathfinder knows OpenClaw received the event, and silent drops are prevented. Pure fire-and-forget would lose events silently if OpenClaw is down. Sync HTTP suits brief planning where the UI blocks on the result.

**Alternatives rejected:**
- Pure fire-and-forget webhook — silent failures if OpenClaw is down.
- Pure sync for all calls — unacceptable latency for event notifications.

---

## Brief Text is Stored in DB

**Decision:** `PlanBrief` records (text + date range) are persisted to the database before calling OpenClaw.

**Rationale:** Brief history gives OpenClaw context across sessions ("last week job hunting, this week interview prep"). Without storage, each call is stateless and OpenClaw cannot detect context drift. Stored history is included in the sync payload as `recent_briefs`.

**Alternatives rejected:**
- Fire-and-forget, no storage — simpler but loses context history across sessions.

---

## Brief Response Contract: OpenClaw Returns JSON, Pathfinder Stores It

**Decision:** OpenClaw sync response is `{ "goals": [...], "tasks": [...] }`. Pathfinder parses the response and writes goals and tasks to the DB. OpenClaw never writes to Pathfinder's DB directly.

**Rationale:** Pathfinder owns the data model. Sync response is the cleanest contract: one request, one response, Pathfinder stores everything. Pathfinder can validate and sanitize before persisting.

**Alternatives rejected:**
- OpenClaw calls back via service token (async) — needs polling or callback endpoint; higher complexity for a flow where the UI already waits.
