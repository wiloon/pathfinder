'use client';
import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getGoals, deleteGoal } from '@/lib/api';
import { GoalCard, Goal } from '@/components/goal-card';
import { GoalEditDialog } from '@/components/goal-edit-dialog';
import { AddGoalDialog } from '@/components/add-goal-dialog';
import { Card, CardContent } from '@/components/ui/card';
import { toast } from 'sonner';

export default function GoalsPage() {
  const queryClient = useQueryClient();
  const [editGoal, setEditGoal] = useState<Goal | null>(null);
  const [editOpen, setEditOpen] = useState(false);

  const { data: goals = [], isLoading } = useQuery<Goal[]>({
    queryKey: ['goals'],
    queryFn: getGoals,
  });

  const deleteMutation = useMutation({
    mutationFn: deleteGoal,
    onSuccess: () => { toast.success('Goal deleted'); queryClient.invalidateQueries({ queryKey: ['goals'] }); },
    onError: () => toast.error('Failed to delete goal'),
  });

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
              onEdit={(g) => { setEditGoal(g); setEditOpen(true); }}
              onDelete={(id) => deleteMutation.mutate(id)}
            />
          ))}
        </div>
      )}

      <GoalEditDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        goal={editGoal}
        onSuccess={() => setEditOpen(false)}
      />
    </div>
  );
}
