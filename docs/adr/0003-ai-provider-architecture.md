# ADR-0003: AI Provider Architecture

**Status:** Accepted
**Date:** Project start / 2026-04-19

---

## MiniMax API as Default AI Backend

**Decision:** Use MiniMax Chat API as the default AI provider. The API is OpenAI-compatible.

**Rationale:** Client requirement. Interface is drop-in compatible with OpenAI SDK patterns, making future provider swaps straightforward.

**Alternatives rejected:**
- OpenAI directly — different provider; same interface pattern; no advantage.

---

## Pluggable AI Provider, Not Hardcoded

**Decision:** The `ai/` package defines a `Provider` interface. The active backend is selected at startup from `config.toml` (`[app] ai_provider = "minimax"` or `"openclaw"`). All callers use package-level functions (`ai.GenerateInitialPlan`, etc.) that delegate to the active provider.

**Rationale:** Pathfinder is a UI + data layer; the decision-making brain is pluggable. User can choose MiniMax (built-in, no extra infra) or OpenClaw (self-hosted, richer reasoning) as the AI backend. Both implement the same `Provider` interface.

**Alternatives rejected:**
- Hardcode MiniMax only — blocks OpenClaw integration.
- Make OpenClaw the only option — breaks standalone usage without OpenClaw.

---

## Brief Planning Provider Selected by Config, Not Fallback Chain

**Decision:** `POST /api/plan/brief` returns HTTP 501 if `ai_provider` is not `"openclaw"`. MiniMax brief implementation is intentionally deferred. No silent fallback.

**Rationale:** Brief planning (free-text → structured goals + tasks) requires OpenClaw's richer multi-step reasoning. Implementing a parallel MiniMax brief path adds complexity with no immediate user value. 501 is the correct response for an explicitly unimplemented path — visible and debuggable.

**Alternatives rejected:**
- Silent fallback to MiniMax — hides misconfiguration; MiniMax cannot produce the same brief output quality.
- Implement both providers now — unnecessary scope at current stage.
