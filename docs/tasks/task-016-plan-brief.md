# Task Spec: #16 — POST /api/plan/brief

**Status:** Pending
**Priority:** P2
**Depends on:** #15 (PlanBrief model ✅)
**Related ADR:** [ADR-0003](../adr/0003-ai-provider-architecture.md), [ADR-0004](../adr/0004-openclaw-integration.md)

---

## Summary

Add `POST /api/plan/brief` endpoint. Accepts a free-text planning brief from the user, saves it to `PlanBrief`, calls the OpenClaw sync URL with full context, parses the structured response, and creates/updates goals and tasks in the database. Returns 501 if `ai_provider` is not `"openclaw"`.

---

## Request

**Auth:** `RequireAuth` (user session)
**Content-Type:** `application/json`

```json
{
  "text":       "string, required, max 2000 chars",
  "start_date": "YYYY-MM-DD, optional, defaults to today",
  "end_date":   "YYYY-MM-DD, optional, defaults to start_date"
}
```

**Validation:**
- `text` required, non-empty, max 2000 chars
- `start_date` / `end_date`: valid YYYY-MM-DD if provided
- `end_date` must be ≥ `start_date`

---

## OpenClaw Sync Payload

Pathfinder POSTs this JSON to `config.openclaw.sync_url`.
Headers: `Authorization: Bearer <openclaw.service_token>`, `Content-Type: application/json`.
Timeout: 30 seconds.

```json
{
  "event": "plan.brief",
  "user_id": "<user_id>",
  "brief": {
    "text": "...",
    "start_date": "YYYY-MM-DD",
    "end_date": "YYYY-MM-DD"
  },
  "active_goals": [
    {
      "id": 1,
      "title": "...",
      "description": "...",
      "weight": 7,
      "tags": ["career"],
      "timeline": "3 months"
    }
  ],
  "recent_briefs": [
    {
      "text": "...",
      "start_date": "YYYY-MM-DD",
      "end_date": "YYYY-MM-DD",
      "created_at": "RFC3339"
    }
  ]
}
```

`recent_briefs` = last 3 `PlanBrief` records by `created_at DESC`, **excluding** the one just saved.

---

## OpenClaw Response Contract

```json
{
  "goals": [
    {
      "title":       "string",
      "description": "string",
      "weight":      7,
      "tags":        ["career"],
      "timeline":    "3 months"
    }
  ],
  "tasks": [
    {
      "title":           "string",
      "description":     "string",
      "goal_title":      "string — must match one of goals[].title",
      "date":            "YYYY-MM-DD",
      "suggested_start": "HH:MM",
      "suggested_end":   "HH:MM"
    }
  ]
}
```

---

## Implementation Steps

1. **Check provider** — if `ai_provider != "openclaw"`, return 501 immediately (do not save brief)
2. **Validate** request body (`text`, `start_date`, `end_date`)
3. **Save** `PlanBrief{UserID, Text, StartDate, EndDate}` to DB
4. **Load context** — active goals for user; last 3 briefs excluding the one just saved
5. **Build sync payload** and POST to `sync_url` (30s timeout, Bearer token)
6. **Parse response** — validate JSON structure; return 502 on timeout or non-2xx; return 500 on unparseable JSON
7. **Upsert goals** — for each goal in response: if a goal with matching `title` exists for this user → update `weight`/`tags`/`timeline`; else create new goal
8. **Create tasks** — for each task in response: find or create `DailyPlan` for `task.date`; create `Task` linked to the plan; match `goal_title` to a goal ID (leave `GoalID` nil if no match)
9. **Return** `{"brief_id": N, "goals_created": N, "goals_updated": N, "tasks_created": N}`

---

## Error Cases

| Condition | HTTP | Body |
|---|---|---|
| `ai_provider != "openclaw"` | 501 | `{"error": "brief planning requires openclaw provider"}` |
| `text` missing or empty | 400 | `{"error": "text is required"}` |
| `text` > 2000 chars | 400 | `{"error": "text exceeds 2000 character limit"}` |
| `end_date` < `start_date` | 400 | `{"error": "end_date must be on or after start_date"}` |
| OpenClaw unreachable / timeout | 502 | `{"error": "AI service unavailable"}` |
| OpenClaw returns non-2xx | 502 | `{"error": "AI service error"}` |
| Response JSON unparseable | 500 | `{"error": "invalid response from AI service"}` |

---

## Tests

Use `httptest.NewServer` to mock the OpenClaw sync endpoint. 6 test cases:

1. **Happy path** — valid request, mock returns goals + tasks, assert DB records created, response counts correct
2. **No openclaw provider** — `ai_provider = "minimax"`, assert 501, assert no `PlanBrief` saved
3. **Missing text** — empty body, assert 400
4. **end_date before start_date** — assert 400
5. **OpenClaw non-2xx** — mock returns 500, assert 502
6. **Invalid JSON response** — mock returns `"not json"`, assert 500

---

## Out of Scope

- MiniMax brief implementation (returns 501, intentional per ADR-0003)
- UI for brief input (task #17)
- Streaming or partial response handling
- Deduplication beyond title-matching for goal upsert
