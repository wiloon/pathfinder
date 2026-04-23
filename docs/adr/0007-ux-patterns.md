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

---

## Tasks Page Uses a Split-Panel Layout (Goals Left, Tasks Right)

**Decision:** The `/tasks` page (`app/tasks/page.tsx`) is redesigned as a two-column layout.
- **Left panel** (`w-72`): Goal list with full CRUD interaction (edit, delete) and click-to-filter. Clicking a Goal filters the right panel to tasks whose `goal_id` matches. Clicking the selected Goal again deselects (returns to "all tasks").
- **Right panel** (`flex-1`): Daily task list, filtered or unfiltered depending on the left-panel selection.

The `GoalCard` component and edit `Dialog` are extracted from `app/goals/page.tsx` into `components/goal-card.tsx` and `components/goal-edit-dialog.tsx` so the same components are shared between `/tasks` and `/goals` without duplication.

**Rationale:** Users need context (which goal does this task serve?) while managing daily tasks. Showing Goals alongside tasks without navigating away reduces context-switching. A left-panel filter is a standard split-panel pattern (VS Code explorer, mail clients) — discoverable and low-friction. Extraction into shared components keeps `/goals` fully functional without copy-pasting logic.

**Alternatives rejected:**
- Keep Goals on a separate `/goals` page only — requires navigation to see goal context while working in `/tasks`; higher friction.
- Embed a goal dropdown/select inside each Task card — adds noise to the task list; does not provide a goal-level overview.
- Full inline goal editing without a dialog — too much vertical space in the left panel; the dialog (already proven in `/goals`) is the right scope for editing.
