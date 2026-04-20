# OpenClaw Integration Design

**Status:** Draft  
**Date:** 2026-04-19  
**Related tasks:** #9, #13, #14, #15, #16 (AGENT_STATE.md)

---

## Overview

Pathfinder is a UI and data layer. When a user configures an OpenClaw endpoint, AI decision-making is delegated to OpenClaw instead of calling MiniMax directly. Two communication patterns are used depending on whether the caller needs an immediate result:

| Pattern | Direction | Used for |
|---|---|---|
| **Synchronous HTTP** | Pathfinder → OpenClaw (waits for response) | Brief planning — user needs result immediately |
| **Webhook** | Pathfinder → OpenClaw (fire-and-forget) | Event notifications: goal created, check-in submitted |
| **Service Token API** | OpenClaw → Pathfinder (authenticated REST calls) | User talks to OpenClaw directly; OpenClaw creates/updates goals and tasks in Pathfinder on their behalf |

When the relevant OpenClaw config keys are absent, Pathfinder falls back to MiniMax. No code path changes for existing users.

---

## Architecture

### Pattern 1: Synchronous Brief Planning

```
User submits brief text (Pathfinder UI)
        │
        ▼
POST /api/plan/brief
        │  save brief to DB
        │  POST <sync_url> (wait...)
        ▼
    OpenClaw
        │  parse brief → goals → tasks
        ▼
    { goals: [...], tasks: [...] }
        │
        ▼
Pathfinder parses response, creates goals + tasks in DB
        │
        ▼
201 response → UI renders goals + tasks
```

### Pattern 2: Webhook Event Notifications

```
User action (goal created, check-in submitted)
        │
        ▼
Pathfinder API
        │  POST webhook (waits for HTTP 2xx delivery ack)
        ├─ non-2xx or timeout ──► user operation fails (return error to user)
        ▼
    OpenClaw returns 2xx immediately
        │  AI background processing (async)
        ▼
    (no callback to Pathfinder)
```

### Pattern 3: OpenClaw → Pathfinder (Service Token)

User interacts with OpenClaw directly (without opening Pathfinder UI). OpenClaw calls Pathfinder's REST API using a service token, creating or updating goals and tasks on the user's behalf.

```
User → OpenClaw (chat / CLI)
        │
        │  interprets intent
        │  selects action
        ▼
POST /api/goals              ← create a goal
POST /api/plan/tasks         ← add a task to today's plan
PATCH /api/tasks/:id         ← update task title / time slot
PUT /api/tasks/:id           ← mark task done / skipped
        │
        │  X-Service-Token: <service_token>
        │  maps to service_user_id in config
        ▼
    Pathfinder DB updated
        │
        ▼
Pathfinder UI reflects changes on next load
```

The service token header bypasses the browser session requirement. Pathfinder maps the token to a configured `service_user_id` and processes the request as that user.

---

## Configuration

Add an `[openclaw]` section to `config.toml`:

```toml
[app]
# ai_provider selects the AI backend: "openclaw" or "minimax".
# Brief planning (POST /api/plan/brief) is only supported with "openclaw".
ai_provider = "openclaw"

[openclaw]
# sync_url: used for brief planning (synchronous call, Pathfinder waits for response).
sync_url = "https://openclaw.example.com/api/plan/brief"

# webhook_url: used for event notifications.
# Pathfinder waits for HTTP 2xx from OpenClaw; failure fails the user's operation.
# Leave empty to skip event notifications.
webhook_url    = "https://openclaw.example.com/webhooks/pathfinder"

# Shared secret for HMAC-SHA256 webhook request signing.
webhook_secret = "your-shared-secret"

# service_token: static bearer token that allows OpenClaw (and agent/curl)
# to call Pathfinder's API without a browser session.
# Generate with: openssl rand -hex 32
service_token = "your-service-token"

# service_user_id: the Pathfinder user ID that service_token authenticates as.
# All API calls made with service_token will operate on this user's data.
service_user_id = "1"
```

---

## Synchronous Brief Planning: Pathfinder → OpenClaw

### Request (Pathfinder calls OpenClaw)

```
POST <sync_url>
Content-Type: application/json

{
  "user_id": "1",
  "timestamp": "2026-04-19T10:00:00+08:00",
  "brief": "I'm job hunting and also studying English",
  "date_range": {
    "start": "2026-04-19",
    "end": "2026-04-20"
  },
  "context": {
    "existing_goals": [
      { "id": 42, "title": "Find a backend engineer job", "type": "primary", "status": "active" }
    ],
    "recent_briefs": [
      { "text": "Started preparing resume", "created_at": "2026-04-18T09:00:00+08:00" }
    ]
  }
}
```

`recent_briefs` contains the last 5 stored briefs, giving OpenClaw context about how the user's situation is evolving.

### Response (OpenClaw returns structured plan)

OpenClaw must respond within 60 seconds. Pathfinder will time out and return an error to the user if exceeded.

```json
{
  "goals": [
    {
      "title": "Find a backend engineer job",
      "description": "Actively apply and prepare for interviews",
      "type": "primary",
      "timeline": "3 months"
    },
    {
      "title": "Improve English proficiency",
      "description": "Daily reading and speaking practice",
      "type": "secondary",
      "timeline": "ongoing"
    }
  ],
  "tasks": [
    {
      "goal_index": 0,
      "title": "Update resume with recent projects",
      "description": "",
      "date": "2026-04-19",
      "suggested_start": "09:00",
      "suggested_end": "10:30"
    },
    {
      "goal_index": 1,
      "title": "Read English news for 20 minutes",
      "description": "",
      "date": "2026-04-19",
      "suggested_start": "20:00",
      "suggested_end": "20:20"
    }
  ]
}
```

`goal_index` links a task to a goal in the `goals` array (0-based). Pathfinder creates goals first, then uses the resulting IDs to set `goal_id` on each task.

### Provider selection

Brief planning uses whichever AI provider is configured under `[app]` (see below). The two providers are mutually exclusive — configure one, not both.

```
app.ai_provider = "openclaw"  ──► POST <openclaw.sync_url>, parse {goals, tasks} response
app.ai_provider = "minimax"   ──► return 501 Not Implemented (brief not yet supported for MiniMax)
```

MiniMax brief planning is deferred. If `ai_provider` is `minimax` and the user calls `POST /api/plan/brief`, Pathfinder returns:

```json
{ "error": "brief planning requires OpenClaw; MiniMax brief is not yet supported" }
```

This is an explicit 501, not a silent fallback.

---

## Webhook: Pathfinder → OpenClaw

### When Pathfinder sends a webhook

| Trigger | Event type |
|---|---|
| User creates a goal | `goal.created` |
| User clicks "Generate Plan" | `plan.generate_requested` |
| User submits evening check-in | `checkin.submitted` |

### Request format

```
POST <webhook_url>
Content-Type: application/json
X-Pathfinder-Signature: sha256=<hmac-hex>
X-Pathfinder-Event: <event_type>
```

The `X-Pathfinder-Signature` header is computed as:

```
HMAC-SHA256(key=webhook_secret, message=raw_request_body)
```

OpenClaw must reject requests where the signature does not match.

### Payload: `goal.created`

```json
{
  "event": "goal.created",
  "user_id": "1",
  "timestamp": "2026-04-19T10:00:00+08:00",
  "data": {
    "goal_id": 42,
    "title": "Learn Go concurrency patterns",
    "description": "...",
    "type": "primary",
    "timeline": "3 months"
  }
}
```

### Payload: `plan.generate_requested`

```json
{
  "event": "plan.generate_requested",
  "user_id": "1",
  "timestamp": "2026-04-19T08:00:00+08:00",
  "data": {
    "date": "2026-04-19",
    "goals": [
      {
        "goal_id": 42,
        "title": "Learn Go concurrency patterns",
        "type": "primary",
        "status": "active"
      }
    ],
    "last_checkin": {
      "date": "2026-04-18",
      "completed": "Finished goroutine basics chapter",
      "blocked": "None",
      "tomorrow_focus": "Practice channel patterns"
    }
  }
}
```

### Payload: `checkin.submitted`

```json
{
  "event": "checkin.submitted",
  "user_id": "1",
  "timestamp": "2026-04-19T22:00:00+08:00",
  "data": {
    "date": "2026-04-19",
    "completed": "Finished 3 tasks",
    "blocked": "Got stuck on select statement",
    "tomorrow_focus": "Review select and timeout patterns",
    "task_summary": {
      "total": 5,
      "done": 3,
      "skipped": 1,
      "pending": 1
    }
  }
}
```

### Pathfinder webhook response expectations

Pathfinder sends the webhook and waits for a response **before returning success to the user**.

| Response | Pathfinder behaviour |
|---|---|
| `2xx` | User operation succeeds; OpenClaw processes AI logic async |
| Non-2xx | User operation fails; Pathfinder returns 502 to the user |
| Timeout (10 s) | User operation fails; Pathfinder returns 504 to the user |

OpenClaw must return `2xx` quickly (before doing any AI processing). All AI work happens in the background after acknowledging receipt.

---

## Fallback Behaviour (Webhook)

```
config.toml has openclaw.webhook_url?
        │
       YES ──► send webhook to OpenClaw
        │
        NO ──► skip (no notification sent)
```

---

## Security Notes

- `webhook_secret` must be a random string of at least 32 characters. Generate with: `openssl rand -hex 32`
- `service_token` must be a random string of at least 32 characters. Generate with: `openssl rand -hex 32`
- Both secrets must be treated as secrets (not committed to version control)
- OpenClaw must always verify `X-Pathfinder-Signature` before processing webhook payloads
- The `sync_url` endpoint on OpenClaw should be protected (e.g., shared secret header or mTLS) — implementation left to OpenClaw
- Service token middleware must check the token in constant time (`crypto/subtle`) to prevent timing attacks
- Pathfinder must return `401` (not `403`) when `X-Service-Token` is absent or invalid, to avoid leaking whether the endpoint exists

---

## Open Questions

| # | Question | Status |
|---|---|---|
| Q1 | Should Pathfinder retry failed webhooks, or is fire-and-forget acceptable for MVP? | **Resolved: webhook failure fails the user's operation immediately; no retry at MVP** |
| Q2 | Should OpenClaw receive the full goal list on every sync call, or only active goals? | **Resolved: send all active goals + last 5 briefs** |
| Q3 | Do we need a `plan.cleared` event so OpenClaw knows when a user manually wipes their plan? | Open |
