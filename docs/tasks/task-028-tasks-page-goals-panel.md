# Task Spec: #28 — Tasks Page Split-Panel: Goals Left, Tasks Right

**Status:** Pending
**Priority:** P2
**Depends on:** #23 (Goal model ✅), #24 (getGoals exposes weight/tags/timeline)
**Related ADR:** [ADR-0007](../adr/0007-ux-patterns.md)

---

## Summary

Redesign `app/tasks/page.tsx` into a two-column split-panel layout.  
- **Left panel** (`w-72 shrink-0`): Goal list with full CRUD interaction (edit, delete) and click-to-filter behaviour.  
- **Right panel** (`flex-1`): Existing daily-task list, filtered to the selected Goal (or all tasks when no Goal is selected).

The Goals page (`app/goals/page.tsx`) is refactored to consume the same extracted components so logic is not duplicated.

---

## Implementation Plan

### Phase 1 — Extract reusable components

#### 1a. `components/goal-card.tsx`

Extract the `GoalCard` function from `app/goals/page.tsx` into a standalone component.

**Exports:**

```typescript
export interface Goal {
  id: number;
  title: string;
  description?: string;
  weight: number;
  tags: string;   // JSON array string
  status: string;
  timeline?: string;
}

export function parseTags(tags: string): string[] { /* JSON.parse with fallback */ }

export function GoalCard({
  goal,
  onEdit,
  onDelete,
  selected,
  onSelect,
}: {
  goal: Goal;
  onEdit: (goal: Goal) => void;
  onDelete: (id: number) => void;
  selected?: boolean;       // highlights the card when true
  onSelect?: (id: number) => void;  // optional — omit on Goals page
}): JSX.Element
```

The `selected` prop adds a ring/border highlight class (`ring-2 ring-primary` or similar).  
The card root `div` calls `onSelect?.(goal.id)` on click when `onSelect` is provided.

---

#### 1b. `components/goal-edit-dialog.tsx`

Extract the edit `Dialog` (form + AI-extract section) from `app/goals/page.tsx` into a standalone component.

**Props:**

```typescript
interface GoalEditDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  goal: Goal | null;
  /** Called after goal update + AI item creation succeed. Parent uses this to close the dialog and invalidate queries. */
  onSuccess: () => void;
}
```

The component is **self-contained**: it owns its own `useForm`, `useMutation` for `updateGoal`, `useMutation` for `createEventQuick`/`createTaskQuick`, and all AI-extract state. The parent passes `onSuccess` to be notified when the full save cycle completes; it does **not** pass a save handler or `isSaving` flag.

---

**AI-extract preview flow (preserve exactly from `app/goals/page.tsx`)**

The dialog contains a secondary section below the main form fields:

1. **Trigger button** — `"⚡ AI 提取事件和任务"` / `"提取中..."` (spinner while `aiParsing`).  
   Calls `parseGoal(description)` using the current Description field value.

2. **Preview panels** — rendered only after a successful extract:
   - **Events panel** (`📅 近期事件`): one checkbox row per extracted event — `title`, `event_date`, optional `note` and `description`.
   - **Tasks panel** (`⚡ 今日准备（立即开始）`): one checkbox row per extracted task — `title`, optional `date` and `description`.
   - All items start **checked** by default.
   - User can uncheck items to exclude them from creation.

3. **Save Changes** — submits the form. The `onEditSubmit` handler runs in order:
   1. `await updateGoal(id, formData)` — saves goal fields.
   2. For each checked event: `await createEventQuick(...)`.
   3. For each checked task: `await createTaskQuick(...)`.
   4. Invalidate `['goals']` and `['today-plan']` queries.
   5. Call `onSuccess()`.
   - Individual item-creation failures show a `toast.error` per item but do **not** abort the sequence.

4. **Dialog close / reopen** — clears `aiEvents`, `aiTasks`, selected sets, and resets the form to the goal's current values.

This is identical behaviour to the current implementation in `app/goals/page.tsx`. The extraction step IS the preview; "Save Changes" is the apply step. No separate confirm step is added.

---

#### 1c. Refactor `app/goals/page.tsx`

- Remove local `GoalCard` function.
- Remove local edit `Dialog` block.
- Import from `@/components/goal-card` and `@/components/goal-edit-dialog`.
- All existing state/mutations/queries remain; only the JSX origins change.
- No behavioural change for users of `/goals`.

---

### Phase 2 — Redesign `app/tasks/page.tsx`

#### Data additions

```typescript
interface Task {
  id: number;
  title: string;
  description?: string;
  suggested_start?: string;
  suggested_end?: string;
  status: string;
  sort_order: number;
  goal_id?: number;   // NEW — already returned by API
}
```

New queries / state at the top of `TodayPage`:

```typescript
import { getGoals, updateGoal, deleteGoal } from '@/lib/api';
import { Goal } from '@/components/goal-card';
import { GoalCard }      from '@/components/goal-card';
import { GoalEditDialog } from '@/components/goal-edit-dialog';

const { data: goals = [] } = useQuery<Goal[]>({
  queryKey: ['goals'],
  queryFn: getGoals,
});

const [selectedGoalId, setSelectedGoalId] = useState<number | null>(null);
const [editGoal, setEditGoal]             = useState<Goal | null>(null);
const [editOpen, setEditOpen]             = useState(false);

const updateGoalMutation = useMutation({
  mutationFn: ({ id, data }: { id: number; data: object }) => updateGoal(id, data),
  onSuccess: () => { toast.success('Goal updated!'); queryClient.invalidateQueries({ queryKey: ['goals'] }); setEditOpen(false); },
  onError: () => toast.error('Failed to update goal'),
});

const deleteGoalMutation = useMutation({
  mutationFn: deleteGoal,
  onSuccess: () => { toast.success('Goal deleted'); queryClient.invalidateQueries({ queryKey: ['goals'] }); setSelectedGoalId(null); },
  onError: () => toast.error('Failed to delete goal'),
});
```

> Note: `updateGoalMutation` is used only for the **delete** and the `GoalEditDialog`'s internal goal-update path calls its own mutation. Keep `deleteGoalMutation` here; remove the `updateGoalMutation` if it is unused after Phase 1 extraction.
```

#### Filter logic

```typescript
const filteredTasks = selectedGoalId
  ? tasks.filter(t => t.goal_id === selectedGoalId)
  : tasks;
```

Pass `filteredTasks` to `SortableContext` and the tasks map instead of `tasks`.

#### Layout

Replace the current `max-w-2xl mx-auto` root wrapper with a two-column flex layout:

```tsx
<div className="flex gap-6 max-w-6xl mx-auto">

  {/* LEFT: Goal panel */}
  <div className="w-72 shrink-0">
    <div className="flex items-center justify-between mb-3">
      <h2 className="text-lg font-semibold">Goals</h2>
      <AddGoalDialog trigger={<Button variant="outline" size="sm">Add</Button>} />
    </div>

    {/* "All" filter chip */}
    <button
      className={`w-full text-left px-3 py-2 rounded-md text-sm mb-2 transition-colors ${
        selectedGoalId === null
          ? 'bg-primary text-primary-foreground'
          : 'text-muted-foreground hover:bg-muted'
      }`}
      onClick={() => setSelectedGoalId(null)}
    >
      All tasks
    </button>

    {/* Goal cards */}
    <div className="space-y-2">
      {goals.map(goal => (
        <GoalCard
          key={goal.id}
          goal={goal}
          selected={selectedGoalId === goal.id}
          onSelect={id => setSelectedGoalId(prev => prev === id ? null : id)}
          onEdit={g => { setEditGoal(g); setEditOpen(true); }}
          onDelete={id => deleteGoalMutation.mutate(id)}
        />
      ))}
    </div>
  </div>

  {/* RIGHT: Tasks panel */}
  <div className="flex-1 min-w-0">
    {/* existing header + view toggle + task list — unchanged except using filteredTasks */}
  </div>

  {/* Edit dialog */}
  <GoalEditDialog
    open={editOpen}
    onOpenChange={setEditOpen}
    goal={editGoal}
    onSuccess={() => {
      setEditOpen(false);
      queryClient.invalidateQueries({ queryKey: ['goals'] });
      queryClient.invalidateQueries({ queryKey: ['today-plan'] });
    }}
  />
</div>
```

Clicking a `GoalCard` again when already selected deselects it (toggle behaviour: `prev === id ? null : id`).

---

## Acceptance Criteria

| # | Criterion |
|---|---|
| AC1 | `/tasks` page renders a two-column layout: left panel contains Goal list, right panel contains task list |
| AC2 | Clicking a Goal card filters the right panel to tasks with matching `goal_id` |
| AC3 | "All tasks" chip shows all tasks regardless of `goal_id` |
| AC4 | Clicking the currently selected Goal card deselects it (returns to "all tasks" view) |
| AC5 | Edit button on a Goal card opens the edit dialog and saves changes on submit |
| AC6 | Delete button on a Goal card removes the goal; selected filter resets to null |
| AC7 | `/goals` page behaviour is unchanged after Phase 1 refactor |
| AC8 | `pnpm lint` and `pnpm build` pass with no errors |

---

## Files Changed

| File | Change |
|---|---|
| `pathfinder-ui/components/goal-card.tsx` | New — extracted `GoalCard`, `Goal`, `parseTags`; adds `selected` + `onSelect` props |
| `pathfinder-ui/components/goal-edit-dialog.tsx` | New — extracted edit `Dialog` with form + AI-extract logic |
| `pathfinder-ui/app/goals/page.tsx` | Refactored — removes local GoalCard + Dialog, imports new components |
| `pathfinder-ui/app/tasks/page.tsx` | Redesigned — split-panel layout, Goal queries + mutations, filter state |

No backend changes required.
