'use client';
import { useQueryClient, useMutation } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { createGoal } from '@/lib/api';
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
  goal_type: z.string().min(1),
  timeline: z.string().optional(),
});
type GoalForm = z.infer<typeof goalSchema>;

const GOAL_TYPES = ['career', 'health', 'education', 'personal', 'other'];

interface AddGoalDialogProps {
  trigger?: React.ReactNode;
  onSuccess?: () => void;
}

export function AddGoalDialog({ trigger, onSuccess }: AddGoalDialogProps) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);

  const { register, handleSubmit, reset, formState: { errors } } = useForm<GoalForm>({
    resolver: zodResolver(goalSchema),
  });

  const createMutation = useMutation({
    mutationFn: createGoal,
    onSuccess: () => {
      toast.success('Goal created!');
      queryClient.invalidateQueries({ queryKey: ['goals'] });
      queryClient.invalidateQueries({ queryKey: ['today-plan'] });
      setOpen(false);
      reset();
      onSuccess?.();
    },
    onError: () => toast.error('Failed to create goal'),
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger ?? <Button>Add Goal</Button>}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add New Goal</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit((data) => createMutation.mutate(data))} className="space-y-4">
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
            <Label>Type *</Label>
            <select
              className="w-full border rounded-md px-3 py-2 text-sm mt-1 bg-background"
              {...register('goal_type')}
            >
              <option value="">Select type...</option>
              {GOAL_TYPES.map(t => (
                <option key={t} value={t} className="capitalize">{t}</option>
              ))}
            </select>
            {errors.goal_type && <p className="text-destructive text-sm mt-1">{errors.goal_type.message}</p>}
          </div>
          <div>
            <Label>Timeline</Label>
            <Input {...register('timeline')} placeholder="e.g., 6 months" className="mt-1" />
          </div>
          <Button type="submit" className="w-full" disabled={createMutation.isPending}>
            {createMutation.isPending ? 'Creating...' : 'Create Goal'}
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  );
}
