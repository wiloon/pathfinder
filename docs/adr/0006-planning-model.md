# ADR-0006: Planning Model

**Status:** Accepted
**Date:** 2026-04-20

---

## Daily Available Hours Stored on UserProfile

**Decision:** Add `DailyAvailableHours float64` to `UserProfile` (default 8.0, valid range 0.5–24.0). Pass this value to `GenerateInitialPlan` and `GeneratePlan` instead of a hardcoded constant.

**Rationale:** Plan quality depends on available time — generating 8 hours of tasks for a user with only 4 hours free creates an unachievable and demoralizing plan. User-level config is the right granularity for a single-user system; no per-day override is needed at MVP. Default 8.0 preserves existing behavior for users who do not configure it.

**Alternatives rejected:**
- Hardcode 8.0 permanently — poor plan quality for users with non-standard schedules.
- Per-day override — complex UI and storage; overkill for MVP.

---

## PlanBrief Stores start_date and end_date

**Decision:** `PlanBrief` model includes `start_date` and `end_date` (YYYY-MM-DD) in addition to `text`.

**Rationale:** The date range is semantic context for a brief — "plan a 3-day sprint starting Monday" targets specific dates. Storing the range enables future OpenClaw prompts to include historical planning windows for context ("last week's sprint covered these dates"). Also useful for debugging which dates a brief was intended to target.

**Alternatives rejected:**
- Store only `text` — simpler model but loses the planning window history that gives OpenClaw cross-session context.
