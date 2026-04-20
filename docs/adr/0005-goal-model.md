# ADR-0005: Goal Data Model

**Status:** Accepted
**Date:** 2026-04-20

---

## Goal Priority as Relative Integer Weight (1–10)

**Decision:** Replace `Goal.Type` (`"primary"` / `"secondary"`) with `Goal.Weight int` (range 1–10, default 5). System normalizes weights to percentages at plan generation time.

Example: goals with weights [8, 4, 2] → time allocation [57%, 29%, 14%].

**Rationale:** Weight is continuous — enables proportional time allocation across any number of goals. The binary `primary`/`secondary` distinction cannot express degrees of priority or produce meaningful percentages. User sets weight per goal; AI receives normalized percentages when generating the daily plan.

**Alternatives rejected:**
- Binary enum (`primary`/`secondary`) — insufficient for multi-goal scheduling; cannot produce proportional time allocation.
- Percentage that must sum to 100 — forces user to manually balance; degrades UX when goals are added or removed.

---

## Goal Category as `tags` JSON Array Column

**Decision:** Add `Goal.Tags string` (stores a JSON array, e.g. `["career","health"]`). No join table. The former `Goal.Type` enum field is removed entirely.

**Rationale:** Tags are multi-valued — a goal can be both `career` and `education`. JSON array in SQLite is sufficient for MVP; no cross-goal tag queries are needed yet. Avoids schema migration complexity of a join table at this stage.

**Alternatives rejected:**
- Single-value enum `type` field — cannot express compound categories.
- `GoalTag` join table — properly normalized but over-engineered for MVP; can be introduced if tag querying becomes necessary.

---

## Goal.Timeline Kept as Nullable String

**Decision:** Keep `Goal.Timeline string`. Empty string means "no deadline / long-term goal". AI extracts timeline from user description if mentioned. User can clear the field to indicate no deadline.

**Rationale:** Timeline is optional semantic context — "I want to achieve this in 3 months" improves AI planning quality. Long-term goals naturally have no fixed deadline. Empty string is the natural nil representation for this field in SQLite/GORM.

**Alternatives rejected:**
- Remove field entirely — loses valuable planning context.
- Mandatory field — forces users to invent arbitrary deadlines for genuinely long-term goals.

---

## PUT /api/goals/:id/primary Removed

**Decision:** Remove the `setPrimaryGoal` endpoint (`PUT /api/goals/:id/primary`) and its frontend counterpart. Goal weight is now adjustable via:
- Standard `UpdateGoal` (`PUT /api/goals/:id`) — for user-initiated changes via UI.
- `PATCH /api/agent/goals/:id` (service token auth) — for OpenClaw to rebalance weights programmatically.

**Rationale:** OpenClaw's primary use case is adjusting goal priorities in response to natural language instructions ("focus more on fitness this week"). The service token endpoint gives OpenClaw direct, explicit access. A binary "set primary" endpoint is redundant and incompatible with the continuous weight model.

**Alternatives rejected:**
- Keep `setPrimaryGoal` as-is — binary, incompatible with weight model.
- User-only weight update — blocks OpenClaw from rebalancing goals on the user's behalf.
