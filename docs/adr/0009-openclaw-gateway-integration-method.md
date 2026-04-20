# ADR-0009: OpenClaw Gateway Integration Method

**Status:** Accepted
**Date:** 2026-04-20

---

## Context

Pathfinder uses OpenClaw as its AI decision-making backend. OpenClaw's Gateway exposes multiple integration surfaces. This ADR records the four candidate methods, the trade-offs between them, and the current choice—so the decision does not need to be re-evaluated from scratch if requirements change.

---

## Candidates

| Method | Communication | OpenClaw changes | OpenClaw pushes to Pathfinder | Session/memory continuity | Integration cost |
|---|---|---|---|---|---|
| **1. OpenAI-compatible HTTP API** (`/v1/chat/completions`) | Sync request/response | None (enable in config) | No | Yes (via `x-openclaw-session-key`) | Very low |
| **2. Webhooks plugin** | Async, Pathfinder polls | None (built-in plugin) | No (polling only) | Via TaskFlow state | Low–medium |
| **3. Tools Invoke API** (`/tools/invoke`) | Sync request/response | None | No | No (stateless per call) | Low |
| **4. Channel Plugin** | Bidirectional, persistent | Yes (new TS extension) | Yes (native push) | Full session context | High |

### Method 1: OpenAI-compatible HTTP API

Pathfinder sends a `POST /v1/chat/completions` request to the OpenClaw Gateway. The Gateway routes the request through the configured agent and returns a structured response. The `model` field selects the agent (`"openclaw"` or `"openclaw/<agentId>"`).

**Pros:**
- Zero OpenClaw code changes; only a config flag.
- Pathfinder already has an OpenAI-compatible client (`ai/ai.go`); switching base URL is sufficient.
- Full agent reasoning, tools, and memory available in the response path.
- Session continuity supported via `x-openclaw-session-key` header.

**Cons:**
- OpenClaw cannot initiate communication to Pathfinder.
- Pathfinder must poll for asynchronous results if needed.

### Method 2: Webhooks Plugin

OpenClaw's built-in Webhooks plugin exposes HTTP routes that bind to TaskFlows. Pathfinder sends an action (`create_flow`, `get_flow`, `get_task_summary`, etc.) to a configured route. TaskFlow tracks multi-step AI work asynchronously.

**Pros:**
- Structured task-flow tracking with persistent state.
- Supports long-running AI workflows.

**Cons:**
- Pathfinder must poll for results; no push back.
- More moving parts: route config, shared secret, TaskFlow lifecycle management.
- Less direct than method 1 for simple request/response planning flows.

### Method 3: Tools Invoke API

Pathfinder calls `POST /tools/invoke` to invoke a specific OpenClaw tool directly, bypassing agent reasoning.

**Pros:**
- Precise, low-latency invocation of a known tool.

**Cons:**
- No agent reasoning or multi-step planning; just a single tool call.
- Stateless—no session or memory continuity.
- Only suitable for narrow, well-defined operations; not for open-ended planning.

### Method 4: Channel Plugin

Pathfinder is implemented as an OpenClaw channel plugin (TypeScript, in `extensions/`), at the same level as Telegram or Discord. OpenClaw can push messages/plans to Pathfinder proactively.

**Pros:**
- Full bidirectional communication.
- OpenClaw can initiate planning reminders, deadline alerts, or task updates.
- Same session/memory/context model as any first-class channel.

**Cons:**
- Requires writing and maintaining a TypeScript extension in the OpenClaw repo.
- Highest integration cost; tightly couples Pathfinder's lifecycle to OpenClaw's plugin API surface.
- Only warranted when proactive push from OpenClaw to Pathfinder is a real requirement.

---

## Decision

**Method 1: OpenAI-compatible HTTP API.**

Enable `gateway.http.endpoints.chatCompletions.enabled: true` in OpenClaw config. Point `pathfinder-api/config.toml` `[ai] base_url` at the Gateway. Set `model = "openclaw"`. No OpenClaw source changes required.

---

## Rationale

1. The current requirement is synchronous: user sets a goal → Pathfinder requests a plan → OpenClaw returns tasks. This is a request/response flow; bidirectional push is not needed yet.
2. Pathfinder already has an OpenAI-compatible client; the changeset is a config swap, not a code rewrite.
3. Method 1 provides full agent reasoning and session continuity—no capability gap for the current scope.
4. Avoids committing to a TypeScript extension before the integration contract is stable.

---

## Trade-offs Accepted

- OpenClaw cannot proactively push plans or reminders to Pathfinder. Scheduled or time-triggered planning must be initiated by Pathfinder (e.g., a cron job calling `/v1/chat/completions`).

---

## Revisit Triggers

Upgrade to **Method 4 (Channel Plugin)** when any of the following become true:

- OpenClaw needs to push a plan or reminder to Pathfinder unprompted (e.g., morning briefing without user action).
- Pathfinder needs real-time task status updates streamed from OpenClaw.
- The integration contract (goals, tasks, check-in payloads) has stabilized and the investment in a typed plugin API is justified.

Upgrade to **Method 2 (Webhooks)** if:

- Planning flows grow beyond a single round-trip (multi-step reasoning that outlasts a single HTTP timeout).
- TaskFlow state tracking becomes useful for Pathfinder's own UI (e.g., showing "planning in progress").

---

## Alternatives Rejected

- **Method 2 (Webhooks)** — Polling overhead and TaskFlow lifecycle management are unnecessary for synchronous planning. Add when async multi-step flows become a real requirement.
- **Method 3 (Tools Invoke)** — Bypasses agent reasoning; insufficient for open-ended goal planning.
- **Method 4 (Channel Plugin)** — Highest cost, requires stable cross-repo contract, and push capability is not currently needed.
