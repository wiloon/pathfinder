'use client';
import { useQueryClient, useMutation } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { createGoal, parseGoal, createEventQuick, createTaskQuick, ParsedGoal, ParsedEvent, ParsedTask } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { toast } from 'sonner';
import { useState } from 'react';

const goalSchema = z.object({
  title: z.string().min(2),
  description: z.string().optional(),
  weight: z.coerce.number().int().min(1).max(10),
  tags: z.string().optional(), // comma-separated, parsed on submit
  timeline: z.string().optional(),
});
type GoalForm = z.infer<typeof goalSchema>;

interface AddGoalDialogProps {
  trigger?: React.ReactNode;
  onSuccess?: () => void;
}

export function AddGoalDialog({ trigger, onSuccess }: AddGoalDialogProps) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  // 'input' = step 1 (free text), 'preview' = step 2 (editable parsed result)
  const [step, setStep] = useState<'input' | 'preview'>('input');
  const [rawInput, setRawInput] = useState('');
  const [isParsing, setIsParsing] = useState(false);
  const [extractedEvents, setExtractedEvents] = useState<ParsedEvent[]>([]);
  const [selectedEvents, setSelectedEvents] = useState<Set<number>>(new Set());
  const [extractedTasks, setExtractedTasks] = useState<ParsedTask[]>([]);
  const [selectedTasks, setSelectedTasks] = useState<Set<number>>(new Set());

  const { register, handleSubmit, reset, setValue, formState: { errors } } = useForm<GoalForm>({
    resolver: zodResolver(goalSchema) as import('react-hook-form').Resolver<GoalForm>,
    defaultValues: { weight: 5 },
  });

  const createMutation = useMutation({
    mutationFn: createGoal,
    onSuccess: () => {
      toast.success('Goal created!');
      queryClient.invalidateQueries({ queryKey: ['goals'] });
      queryClient.invalidateQueries({ queryKey: ['today-plan'] });
      handleClose();
      onSuccess?.();
    },
    onError: () => toast.error('Failed to create goal'),
  });

  const handleClose = () => {
    setOpen(false);
    setStep('input');
    setRawInput('');
    setExtractedEvents([]);
    setSelectedEvents(new Set());
    setExtractedTasks([]);
    setSelectedTasks(new Set());
    reset();
  };

  const handleParse = async () => {
    if (!rawInput.trim()) return;
    setIsParsing(true);
    try {
      const parsed: ParsedGoal = await parseGoal(rawInput.trim());
      setValue('title', parsed.title);
      setValue('description', parsed.description || '');
      setValue('weight', parsed.weight);
      setValue('tags', parsed.tags.join(', '));
      setValue('timeline', parsed.timeline || '');
      const events = (parsed.extracted_events || []).filter(e => e.event_date);
      setExtractedEvents(events);
      setSelectedEvents(new Set(events.map((_, i) => i)));
      const tasks = parsed.extracted_tasks || [];
      setExtractedTasks(tasks);
      setSelectedTasks(new Set(tasks.map((_, i) => i)));
      setStep('preview');
    } catch {
      toast.error('AI parsing failed. You can fill in the details manually.');
      setStep('preview');
    } finally {
      setIsParsing(false);
    }
  };

  const onSubmit = async (data: GoalForm) => {
    const tags = data.tags
      ? data.tags.split(',').map(t => t.trim()).filter(Boolean)
      : [];
    await createMutation.mutateAsync({ ...data, tags });
    // Create confirmed extracted events.
    for (const [i, ev] of extractedEvents.entries()) {
      if (selectedEvents.has(i) && ev.event_date) {
        try {
          await createEventQuick({ title: ev.title, description: ev.description, event_date: ev.event_date });
        } catch {
          toast.error(`创建事件失败: ${ev.title}`);
        }
      }
    }
    // Create confirmed prep tasks.
    for (const [i, task] of extractedTasks.entries()) {
      if (selectedTasks.has(i)) {
        try {
          await createTaskQuick({ title: task.title, description: task.description, date: task.date || undefined });
        } catch {
          toast.error(`创建任务失败: ${task.title}`);
        }
      }
    }
  };

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) handleClose(); else setOpen(true); }}>
      <DialogTrigger asChild>
        {trigger ?? <Button>Add Goal</Button>}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{step === 'input' ? 'Add New Goal' : 'Review & Edit Goal'}</DialogTitle>
        </DialogHeader>

        {step === 'input' ? (
          <div className="space-y-4">
            <div>
              <Label>Describe your goal</Label>
              <Textarea
                value={rawInput}
                onChange={e => setRawInput(e.target.value)}
                placeholder="e.g. I want to get fit and run a 5K by the end of summer"
                className="mt-1"
                rows={4}
                maxLength={500}
              />
              <p className="text-muted-foreground text-xs mt-1">{rawInput.length}/500</p>
            </div>
            <div className="flex gap-2">
              <Button
                className="flex-1"
                onClick={handleParse}
                disabled={!rawInput.trim() || isParsing}
              >
                {isParsing ? 'Parsing...' : 'Parse with AI'}
              </Button>
              <Button variant="outline" onClick={() => setStep('preview')}>
                Skip
              </Button>
            </div>
          </div>
        ) : (
          <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
            <div>
              <Label>Title *</Label>
              <Input {...register('title')} placeholder="Goal title" className="mt-1" />
              {errors.title && <p className="text-destructive text-sm mt-1">{errors.title.message}</p>}
            </div>
            <div>
              <Label>Description</Label>
              <Textarea {...register('description')} placeholder="Description" className="mt-1" rows={3} />
            </div>
            <div>
              <Label>Priority (1–10)</Label>
              <Input {...register('weight')} type="number" min={1} max={10} className="mt-1" />
              {errors.weight && <p className="text-destructive text-sm mt-1">{errors.weight.message}</p>}
            </div>
            <div>
              <Label>Tags</Label>
              <Input {...register('tags')} placeholder="e.g. career, health" className="mt-1" />
              <p className="text-muted-foreground text-xs mt-1">Comma-separated</p>
            </div>
            <div>
              <Label>Timeline</Label>
              <Input {...register('timeline')} placeholder="e.g. 3 months (leave empty for long-term)" className="mt-1" />
            </div>
            {extractedEvents.length > 0 && (
              <div className="border-t pt-3 space-y-2">
                <Label className="text-sm font-medium">📅 近期事件（AI 提取）</Label>
                {extractedEvents.map((ev, i) => (
                  <div key={i} className="flex items-start gap-2 py-1">
                    <input
                      type="checkbox"
                      className="mt-0.5"
                      checked={selectedEvents.has(i)}
                      onChange={() => {
                        setSelectedEvents(prev => {
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
            {extractedTasks.length > 0 && (
              <div className="border-t pt-3 space-y-2">
                <Label className="text-sm font-medium">⚡ 今日准备（立即开始）</Label>
                {extractedTasks.map((task, i) => (
                  <div key={i} className="flex items-start gap-2 py-1">
                    <input
                      type="checkbox"
                      className="mt-0.5"
                      checked={selectedTasks.has(i)}
                      onChange={() => {
                        setSelectedTasks(prev => {
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
            <div className="flex gap-2">
              <Button type="button" variant="outline" onClick={() => setStep('input')}>
                Back
              </Button>
              <Button type="submit" className="flex-1" disabled={createMutation.isPending}>
                {createMutation.isPending ? 'Creating...' : 'Create Goal'}
              </Button>
            </div>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
