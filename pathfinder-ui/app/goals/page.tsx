'use client';
import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { getGoals, updateGoal, deleteGoal, setPrimaryGoal } from '@/lib/api';
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
  goal_type: z.string().min(1),
  status: z.string().optional(),
  timeline: z.string().optional(),
});
type GoalForm = z.infer<typeof goalSchema>;

interface Goal {
  id: number;
  title: string;
  description?: string;
  goal_type: string;
  is_primary: boolean;
  status: string;
  timeline?: string;
}

const GOAL_TYPES = ['career', 'health', 'education', 'personal', 'other'];
const GOAL_STATUSES = ['active', 'paused', 'completed', 'abandoned'];

function GoalCard({ goal, onEdit, onDelete, onSetPrimary }: {
  goal: Goal;
  onEdit: (goal: Goal) => void;
  onDelete: (id: number) => void;
  onSetPrimary: (id: number) => void;
}) {
  return (
    <Card className={goal.is_primary ? 'border-primary' : ''}>
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between gap-2">
          <CardTitle className="text-base leading-tight">{goal.title}</CardTitle>
          <div className="flex gap-1 shrink-0">
            <Badge variant={goal.is_primary ? 'default' : 'outline'} className="capitalize">
              {goal.is_primary ? 'Primary' : 'Secondary'}
            </Badge>
            <Badge variant="outline" className="capitalize">{goal.status}</Badge>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {goal.description && <p className="text-sm text-muted-foreground mb-3">{goal.description}</p>}
        <div className="flex flex-wrap gap-2 text-sm text-muted-foreground mb-4">
          <span className="capitalize">📁 {goal.goal_type}</span>
          {goal.timeline && <span>⏱ {goal.timeline}</span>}
        </div>
        <div className="flex gap-2 flex-wrap">
          {!goal.is_primary && (
            <Button size="sm" variant="outline" onClick={() => onSetPrimary(goal.id)}>Set Primary</Button>
          )}
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

  const { data: goals = [], isLoading } = useQuery<Goal[]>({
    queryKey: ['goals'],
    queryFn: getGoals,
  });

  const { register: regEdit, handleSubmit: handleEdit, setValue: setEditValue } = useForm<GoalForm>({ resolver: zodResolver(goalSchema) });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: GoalForm }) => updateGoal(id, data),
    onSuccess: () => { toast.success('Goal updated!'); queryClient.invalidateQueries({ queryKey: ['goals'] }); setEditOpen(false); },
    onError: () => toast.error('Failed to update goal'),
  });

  const deleteMutation = useMutation({
    mutationFn: deleteGoal,
    onSuccess: () => { toast.success('Goal deleted'); queryClient.invalidateQueries({ queryKey: ['goals'] }); },
    onError: () => toast.error('Failed to delete goal'),
  });

  const setPrimaryMutation = useMutation({
    mutationFn: setPrimaryGoal,
    onSuccess: () => { toast.success('Primary goal updated!'); queryClient.invalidateQueries({ queryKey: ['goals'] }); },
    onError: () => toast.error('Failed to set primary goal'),
  });

  const openEdit = (goal: Goal) => {
    setEditGoal(goal);
    setEditValue('title', goal.title);
    setEditValue('description', goal.description || '');
    setEditValue('goal_type', goal.goal_type);
    setEditValue('status', goal.status);
    setEditValue('timeline', goal.timeline || '');
    setEditOpen(true);
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
              onSetPrimary={(id) => setPrimaryMutation.mutate(id)}
            />
          ))}
        </div>
      )}

      {/* Edit Dialog */}
      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent>
          <DialogHeader><DialogTitle>Edit Goal</DialogTitle></DialogHeader>
          {editGoal && (
            <form onSubmit={handleEdit((data) => updateMutation.mutate({ id: editGoal.id, data }))} className="space-y-4">
              <div>
                <Label>Title *</Label>
                <Input {...regEdit('title')} className="mt-1" />
              </div>
              <div>
                <Label>Description</Label>
                <Textarea {...regEdit('description')} className="mt-1" rows={3} />
              </div>
              <div>
                <Label>Type *</Label>
                <select className="w-full border rounded-md px-3 py-2 text-sm mt-1 bg-background" {...regEdit('goal_type')}>
                  {GOAL_TYPES.map(t => <option key={t} value={t} className="capitalize">{t}</option>)}
                </select>
              </div>
              <div>
                <Label>Status</Label>
                <select className="w-full border rounded-md px-3 py-2 text-sm mt-1 bg-background" {...regEdit('status')}>
                  {GOAL_STATUSES.map(s => <option key={s} value={s} className="capitalize">{s}</option>)}
                </select>
              </div>
              <div>
                <Label>Timeline</Label>
                <Input {...regEdit('timeline')} className="mt-1" />
              </div>
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
