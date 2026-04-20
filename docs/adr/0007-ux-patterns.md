# ADR-0007: UX and API Design Patterns

**Status:** Accepted
**Date:** 2026-04-20

---

## AI-Parsed Goal Creation Requires Two-Step Preview

**Decision:** `AddGoalDialog` uses a two-step flow:
- **Step 1:** User enters a free-text description (single textarea, max 500 chars).
- **Step 2:** User reviews and edits AI-parsed structured fields (title, description, weight, tags, timeline) before confirming creation.

Creation only occurs after Step 2 confirmation.

**Rationale:** Reduces cognitive load at input — user writes natural language instead of filling multiple form fields. Preserves user control — AI misinterpretation can be corrected before data is persisted. One-step direct create is faster but loses user trust if the AI misreads intent; a bad first goal could corrupt the initial plan.

**Alternatives rejected:**
- One-step direct create — faster UX but no correction opportunity; AI errors silently create bad data.
- Keep the current multi-field form — high cognitive friction; no AI assistance; removed by this decision.

---

## POST /api/goals/parse is a Stateless, Read-Only Endpoint

**Decision:** `POST /api/goals/parse` accepts `raw_input`, calls `ai.ParseGoalText`, and returns a `ParsedGoal` JSON response. It does **not** write to the database.

**Rationale:** Parsing and creating are separate concerns. A stateless parse endpoint allows the UI to call it multiple times (e.g., user edits the input and re-parses) without side effects or partial data in the database. Goal creation still goes through the existing `POST /api/goals` path, keeping the data layer unchanged.

**Alternatives rejected:**
- Parse-and-create in one endpoint — couples two concerns; prevents the preview/re-parse flow; creates junk data if the user abandons after parsing.
