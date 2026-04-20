# Task Spec: #22 — AddGoalDialog Two-Step Refactor

**Status:** Pending
**Priority:** P2
**Depends on:** #21 (parseGoal API function), #23 (Goal model ✅)
**Related ADR:** [ADR-0007](../adr/0007-ux-patterns.md)

---

## Summary

Refactor `components/add-goal-dialog.tsx` from a single multi-field form into a two-step AI-assisted flow. Step 1: user enters a free-text description. Step 2: user reviews and edits AI-parsed structured fields before confirming creation.

---

## Frontend Types

```typescript
interface ParsedGoal {
  title:       string;
  description: string;
  weight:      number;   // 1–10
  tags:        string[];
  timeline:    string;   // empty = long-term
}

type DialogStep = 1 | 2;
```

---

## Component State

Managed with `useState` alongside `useForm` (Step 2 fields only):

```typescript
const [step, setStep]             = useState<DialogStep>(1);
const [rawInput, setRawInput]     = useState('');
const [parsedGoal, setParsedGoal] = useState<ParsedGoal | null>(null);
const [isParsing, setIsParsing]   = useState(false);
```

---

## Step 1 UI

| Element | Details |
|---|---|
| `Textarea` | `value={rawInput}`, `onChange`, `maxLength={500}`, `rows={4}` |
| Character counter | `{rawInput.length}/500` displayed below textarea |
| Placeholder | `"Describe your goal... e.g. I want to find a backend engineering job in 3 months"` |
| Submit button | Label: `"Parse with AI"` / `"Parsing…"` (spinner when `isParsing`); disabled when `rawInput.trim().length < 5 \|\| isParsing` |

**On submit:**
1. `setIsParsing(true)`
2. Call `parseGoal(rawInput)` (from `lib/api.ts`)
3. On success: `setParsedGoal(result)`, `setStep(2)`, `reset({ ...result, tags: result.tags.join(', ') })`
4. On error: `toast.error('Failed to parse goal — please try again')`, stay on Step 1
5. `setIsParsing(false)` in both cases (finally)

---

## Step 2 UI

Five editable fields pre-populated from `parsedGoal`:

| Field | Component | Validation | Note |
|---|---|---|---|
| Title | `Input` | required, min 2, max 200 | |
| Description | `Textarea rows={3}` | optional | |
| Weight | `Input type="number"` | min=1, max=10, integer | Label: `"Priority (1–10)"` |
| Tags | `Input` | optional | Comma-separated string; split on confirm |
| Timeline | `Input` | optional | Placeholder: `"e.g. 3 months — leave empty for long-term"` |

- **Back button** — `onClick={() => setStep(1)}`; rawInput is preserved in state (user can re-edit)
- **"Create Goal" button** — submits form; `disabled={createMutation.isPending}`

---

## Zod Schema (Step 2)

```typescript
const goalSchema = z.object({
  title:       z.string().min(2).max(200),
  description: z.string().optional(),
  weight:      z.number().int().min(1).max(10).default(5),
  tags:        z.string().optional(),     // comma-separated; split before POST
  timeline:    z.string().optional(),
});
type GoalForm = z.infer<typeof goalSchema>;
```

---

## On Confirm (Step 2 Submit)

```typescript
const payload = {
  title:       data.title,
  description: data.description ?? '',
  weight:      data.weight,
  tags:        data.tags
                 ? data.tags.split(',').map(t => t.trim()).filter(Boolean)
                 : [],
  timeline:    data.timeline ?? '',
};
createMutation.mutate(payload);
// createGoal() POSTs application/json to POST /api/goals
```

---

## Dialog Reset

On dialog close (`onOpenChange(false)`) or successful creation:
1. `setStep(1)`
2. `setRawInput('')`
3. `setParsedGoal(null)`
4. `reset()` (react-hook-form)
5. `setOpen(false)` (on success only)

---

## Error Handling

| Scenario | Behavior |
|---|---|
| Parse API call fails | `toast.error('Failed to parse goal — please try again')`, stay on Step 1, `setIsParsing(false)` |
| Create API call fails | `toast.error('Failed to create goal')`, stay on Step 2 (user can retry or adjust fields) |

---

## Out of Scope

- Re-parse button on Step 2 (user edits Step 2 fields directly instead)
- File attachment support in this dialog (existing attachment flow is separate)
- Saving draft state across dialog close/open
- Optimistic UI for goal creation
