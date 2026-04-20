'use client';
import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { getGoals, updateGoal, deleteGoal, parseGoal, createEventQuick, createTaskQuick, ParsedEvent, ParsedTask } from '@/lib/api';
import { AddGoalDialog } from '@/components/add-goal-dialog';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { toast } from 'sonner';

const goalSchema = z.object({
  title: z.string().min(2),
  description: z.string().optional(),
  weight: z.coerce.number().int().min(1).max(10),
  tags: z.string().optional(),
  status: z.string().optional(),
  timeline: z.string().optional(),
});
type GoalForm = z.infer<typeof goalSchema>;

interface Goal {
  id: number;
  title: string;
  description?: string;
  weight: number;
  tags: string; // JSON array string from API
  status: string;
  timeline?: string;
}

const GOAL_STATUSES = ['active', 'paused', 'completed'];

function parseTags(tags: string): string[] {
  try {
    const parsed = JSON.parse(tags);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function GoalCard({ goal, onEdit, onDelete }: {
  goal: Goal;
  onEdit: (goal: Goal) => void;
  onDelete: (id: number) => void;
}) {
  const tags = parseTags(goal.tags);
  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between gap-2">
          <CardTitle className="text-base leading-tight">{goal.title}</CardTitle>
          <div className="flex gap-1 shrink-0">
            <Badge variant="outline">P{goal.weight}</Badge>
            <Badge variant="outline" className="capitalize">{goal.status}</Badge>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {goal.description && <p className="text-sm text-muted-foreground mb-3">{goal.description}</p>}
        <div className="flex flex-wrap gap-1 mb-3">
          {tags.map(tag => (
            <Badge key={tag} variant="secondary" className="capitalize text-xs">{tag}</Badge>
          ))}
        </div>
        <div className="flex flex-wrap gap-2 text-sm text-muted-foreground mb-4">
          {goal.timeline && <span>⏱ {goal.timeline}</span>}
        </div>
        <div className="flex gap-2 flex-wrap">
          <Button size="sm" variant="outline" onClick={() => onEdit(goal)}>Edit</Button>
          <Button size="sm" variant="destructive" onClick={() => onDelete(goal.id)}>Delete</Button>
        </div>
      </CardContent>
    </Card>
  );
}

export default function GoalsPage() {
  const queryClient = useQueryClient();
  const [editGoal, setEditGoal] = useState<Goal | null>(null);
  const [editOpen, setEditOpen] = useState(false);
  const [aiParsing, setAiParsing] = useState(false);
  const [aiEvents, setAiEvents] = useState<ParsedEvent[]>([]);
  const [selectedAiEvents, setSelectedAiEvents] = useState<Set<number>>(new Set());
  const [aiTasks, setAiTasks] = useState<ParsedTask[]>([]);
  const [selectedAiTasks, setSelectedAiTasks] = useState<Set<number>>(new Set());

  const { data: goals = [], isLoading } = useQuery<Goal[]>({
    queryKey: ['goals'],
    queryFn: getGoals,
  });

  const { register: regEdit, handleSubmit: handleEdit, setValue: setEditValue, getValues: getEditValues } = useForm<GoalForm>({
    resolver: zodResolver(goalSchema) as import('react-hook-form').Resolver<GoalForm>,
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: GoalForm }) => {
      const tags = data.tags
        ? data.tags.split(',').map(t => t.trim()).filter(Boolean)
        : [];
      return updateGoal(id, { ...data, tags });
    },
    onSuccess: () => {
      toast.success('Goal updated!');
      queryClient.invalidateQueries({ queryKey: ['goals'] });
      setEditOpen(false);
    },
    onError: () => toast.error('Failed to update goal'),
  });

  const deleteMutation = useMutation({
    mutationFn: deleteGoal,
    onSuccess: () => { toast.success('Goal deleted'); queryClient.invalidateQueries({ queryKey: ['goals'] }); },
    onError: () => toast.error('Failed to delete goal'),
  });

  const openEdit = (goal: Goal) => {
    setEditGoal(goal);
    setEditValue('title', goal.title);
    setEditValue('description', goal.description || '');
    setEditValue('weight', goal.weight);
    setEditValue('tags', parseTags(goal.tags).join(', '));
    setEditValue('status', goal.status);
    setEditValue('timeline', goal.timeline || '');
    setAiEvents([]);
    setSelectedAiEvents(new Set());
    setAiTasks([]);
    setSelectedAiTasks(new Set());
    setEditOpen(true);
  };

  const handleAiExtract = async () => {
    const description = getEditValues('description') || getEditValues('title') || '';
    if (!description.trim()) { toast.info('请先填写 Description'); return; }
    setAiParsing(true);
    try {
      const parsed = await parseGoal(description.trim());
      const events = (parsed.extracted_events || []).filter(e => e.event_date);
      setAiEvents(events);
      setSelectedAiEvents(new Set(events.map((_, i) => i)));
      const tasks = parsed.extracted_tasks || [];
      setAiTasks(tasks);
      setSelectedAiTasks(new Set(tasks.map((_, i) => i)));
      if (events.length === 0 && tasks.length === 0) toast.info('未提取到具体事件或任务');
    } catch {
      toast.error('AI 提取失败');
    } finally {
      setAiParsing(false);
    }
  };

  const onEditSubmit = async (data: GoalForm) => {
    await updateMutation.mutateAsync({ id: editGoal!.id, data });
    for (const [i, ev] of aiEvents.entries()) {
      if (selectedAiEvents.has(i) && ev.event_date) {
        try {
          await createEventQuick({ title: ev.title, description: ev.description, event_date: ev.event_date });
        } catch {
          toast.error(`创建事件失败: ${ev.title}`);
        }
      }
    }
    for (const [i, task] of aiTasks.entries()) {
      if (selectedAiTasks.has(i)) {
        try {
          await createTaskQuick({ title: task.title, description: task.description, date: task.date || undefined });
        } catch {
          toast.error(`创建任务失败: ${task.title}`);
        }
      }
    }
    queryClient.invalidateQueries({ queryKey: ['today-plan'] });
  };

  if (isLoading) return <div className="flex justify-center py-12"><div className="text-muted-foreground">Loading goals...</div></div>;

  return (
    <div className="max-w-4xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">Goals</h1>
        <AddGoalDialog />
      </div>

      {goals.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center">
            <p className="text-muted-foreground mb-4">No goals yet. Add your first goal to get started!</p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {goals.map((goal) => (
            <GoalCard
              key={goal.id}
              goal={goal}
              onEdit={openEdit}
              onDelete={(id) => deleteMutation.mutate(id)}
            />
          ))}
        </div>
      )}

      {/* Edit Dialog */}
      <Dialog open={editOpen} onOpenChange={(open) => { setEditOpen(open); if (!open) { setAiEvents([]); setAiTasks([]); setSelectedAiEvents(new Set()); setSelectedAiTasks(new Set()); } }}>
        <DialogContent>
          <DialogHeader><DialogTitle>Edit Goal</DialogTitle></DialogHeader>
          {editGoal && (
            <form onSubmit={handleEdit(onEditSubmit)} className="space-y-4">
              <div>
                <Label>Title *</Label>
                <Input {...regEdit('title')} className="mt-1" />
              </div>
              <div>
                <Label>Description</Label>
                <Textarea {...regEdit('description')} className="mt-1" rows={3} />
              </div>
              <div>
                <Label>Priority (1–10)</Label>
                <Input {...regEdit('weight')} type="number" min={1} max={10} className="mt-1" />
              </div>
              <div>
                <Label>Tags</Label>
                <Input {...regEdit('tags')} placeholder="e.g. career, health" className="mt-1" />
                <p className="text-muted-foreground text-xs mt-1">Comma-separated</p>
              </div>
              <div>
                <Label>Status</Label>
                <select className="w-full border rounded-md px-3 py-2 text-sm mt-1 bg-background" {...regEdit('status')}>
                  {GOAL_STATUSES.map(s => <option key={s} value={s} className="capitalize">{s}</option>)}
                </select>
              </div>
              <div>
                <Label>Timeline</Label>
                <Input {...regEdit('timeline')} placeholder="e.g. 3 months (leave empty for long-term)" className="mt-1" />
              </div>
              <div className="border-t pt-3">
                <Button type="button" variant="outline" size="sm" onClick={handleAiExtract} disabled={aiParsing}>
                  {aiParsing ? '提取中...' : '⚡ AI 提取事件和任务'}
                </Button>
                <p className="text-xs text-muted-foreground mt-1">根据上方 Description 提取近期事件和今日准备任务</p>
              </div>
              {aiEvents.length > 0 && (
                <div className="border rounded-md p-3 space-y-2">
                  <p className="text-xs font-medium">📅 近期事件：</p>
                  {aiEvents.map((ev, i) => (
                    <div key={i} className="flex items-start gap-2">
                      <input
                        type="checkbox"
                        className="mt-0.5"
                        checked={selectedAiEvents.has(i)}
                        onChange={() => {
                          setSelectedAiEvents(prev => {
                            const next = new Set(prev);
                            if (next.has(i)) next.delete(i); else next.add(i);
                            return next;
                          });
                        }}
                      />
                      <div>
                        <span className="text-sm font-medium">{ev.title}</span>
                        <span className="text-xs text-muted-foreground ml-2">{ev.event_date}{ev.note ? ` (${ev.note})` : ''}</span>
                        {ev.description && <p className="text-xs text-muted-foreground">{ev.description}</p>}
                      </div>
                    </div>
                  ))}
                </div>
              )}
              {aiTasks.length > 0 && (
                <div className="border rounded-md p-3 space-y-2">
                  <p className="text-xs font-medium">⚡ 今日准备（立即开始）：</p>
                  {aiTasks.map((task, i) => (
                    <div key={i} className="flex items-start gap-2">
                      <input
                        type="checkbox"
                        className="mt-0.5"
                        checked={selectedAiTasks.has(i)}
                        onChange={() => {
                          setSelectedAiTasks(prev => {
                            const next = new Set(prev);
                            if (next.has(i)) next.delete(i); else next.add(i);
                            return next;
                          });
                        }}
                      />
                      <div>
                        <span className="text-sm font-medium">{task.title}</span>
                        <span className="text-xs text-muted-foreground ml-2">{task.date}</span>
                        {task.description && <p className="text-xs text-muted-foreground">{task.description}</p>}
                      </div>
                    </div>
                  ))}
                </div>
              )}
              <Button type="submit" className="w-full" disabled={updateMutation.isPending}>
                {updateMutation.isPending ? 'Saving...' : 'Save Changes'}
              </Button>
            </form>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
