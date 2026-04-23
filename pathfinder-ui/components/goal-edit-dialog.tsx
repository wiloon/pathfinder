'use client';
import { useState, useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { updateGoal, parseGoal, createEventQuick, createTaskQuick, ParsedEvent, ParsedTask } from '@/lib/api';
import { Goal, parseTags } from '@/components/goal-card';
import { Button } from '@/components/ui/button';
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

const GOAL_STATUSES = ['active', 'paused', 'completed'];

interface GoalEditDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  goal: Goal | null;
  /** Called after goal update + AI item creation complete. Parent uses this to close the dialog. */
  onSuccess: () => void;
}

export function GoalEditDialog({ open, onOpenChange, goal, onSuccess }: GoalEditDialogProps) {
  const queryClient = useQueryClient();
  const [aiParsing, setAiParsing] = useState(false);
  const [aiEvents, setAiEvents] = useState<ParsedEvent[]>([]);
  const [selectedAiEvents, setSelectedAiEvents] = useState<Set<number>>(new Set());
  const [aiTasks, setAiTasks] = useState<ParsedTask[]>([]);
  const [selectedAiTasks, setSelectedAiTasks] = useState<Set<number>>(new Set());

  const { register, handleSubmit, reset, getValues } = useForm<GoalForm>({
    resolver: zodResolver(goalSchema) as import('react-hook-form').Resolver<GoalForm>,
  });

  // Reset form and clear AI state whenever the dialog opens or the goal changes.
  // `reset` is stable across renders; goal.id + open are the meaningful deps.
  useEffect(() => {
    if (!open || !goal) return;
    reset({
      title: goal.title,
      description: goal.description || '',
      weight: goal.weight,
      tags: parseTags(goal.tags).join(', '),
      status: goal.status,
      timeline: goal.timeline || '',
    });
    setAiEvents([]);
    setSelectedAiEvents(new Set());
    setAiTasks([]);
    setSelectedAiTasks(new Set());
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [goal?.id, open]);

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: GoalForm }) => {
      const tags = data.tags
        ? data.tags.split(',').map(t => t.trim()).filter(Boolean)
        : [];
      return updateGoal(id, { ...data, tags });
    },
  });

  const handleAiExtract = async () => {
    const description = getValues('description') || getValues('title') || '';
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
    if (!goal) return;
    try {
      await updateMutation.mutateAsync({ id: goal.id, data });
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
      queryClient.invalidateQueries({ queryKey: ['goals'] });
      queryClient.invalidateQueries({ queryKey: ['today-plan'] });
      toast.success('Goal updated!');
      onSuccess();
    } catch {
      toast.error('Failed to update goal');
    }
  };

  const handleOpenChange = (isOpen: boolean) => {
    if (!isOpen) {
      setAiEvents([]);
      setAiTasks([]);
      setSelectedAiEvents(new Set());
      setSelectedAiTasks(new Set());
    }
    onOpenChange(isOpen);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader><DialogTitle>Edit Goal</DialogTitle></DialogHeader>
        {goal && (
          <form onSubmit={handleSubmit(onEditSubmit)} className="space-y-4">
            <div>
              <Label>Title *</Label>
              <Input {...register('title')} className="mt-1" />
            </div>
            <div>
              <Label>Description</Label>
              <Textarea {...register('description')} className="mt-1" rows={3} />
            </div>
            <div>
              <Label>Priority (1–10)</Label>
              <Input {...register('weight')} type="number" min={1} max={10} className="mt-1" />
            </div>
            <div>
              <Label>Tags</Label>
              <Input {...register('tags')} placeholder="e.g. career, health" className="mt-1" />
              <p className="text-muted-foreground text-xs mt-1">Comma-separated</p>
            </div>
            <div>
              <Label>Status</Label>
              <select className="w-full border rounded-md px-3 py-2 text-sm mt-1 bg-background" {...register('status')}>
                {GOAL_STATUSES.map(s => <option key={s} value={s} className="capitalize">{s}</option>)}
              </select>
            </div>
            <div>
              <Label>Timeline</Label>
              <Input {...register('timeline')} placeholder="e.g. 3 months (leave empty for long-term)" className="mt-1" />
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
  );
}
