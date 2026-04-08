'use client';
import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
} from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { getTodayPlan, updateTask, generatePlan } from '@/lib/api';
import { AddGoalDialog } from '@/components/add-goal-dialog';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { toast } from 'sonner';
import Link from 'next/link';

interface Task {
  id: number;
  title: string;
  description?: string;
  suggested_start?: string;
  suggested_end?: string;
  status: string;
  sort_order: number;
}

interface TodayPlan {
  date: string;
  tasks: Task[];
}

function SortableTask({ task, onStatusChange }: { task: Task; onStatusChange: (id: number, status: string) => void }) {
  const { attributes, listeners, setNodeRef, transform, transition } = useSortable({ id: task.id });
  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  const statusColors: Record<string, string> = {
    pending: 'bg-yellow-100 text-yellow-800',
    done: 'bg-green-100 text-green-800',
    skipped: 'bg-gray-100 text-gray-600',
    in_progress: 'bg-blue-100 text-blue-800',
  };

  return (
    <div ref={setNodeRef} style={style} className={`flex items-start gap-3 p-4 rounded-lg border bg-card transition-opacity ${task.status === 'done' ? 'opacity-60' : ''}`}>
      <button {...attributes} {...listeners} className="mt-1 cursor-grab text-muted-foreground hover:text-foreground">
        ⠿
      </button>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 flex-wrap">
          <span className={`font-medium ${task.status === 'done' ? 'line-through text-muted-foreground' : ''}`}>
            {task.title}
          </span>
          <Badge className={statusColors[task.status] || 'bg-gray-100'} variant="outline">
            {task.status}
          </Badge>
        </div>
        {task.description && <p className="text-sm text-muted-foreground mt-1">{task.description}</p>}
        {(task.suggested_start || task.suggested_end) && (
          <p className="text-xs text-muted-foreground mt-1">
            {task.suggested_start && task.suggested_start.slice(0, 5)}
            {task.suggested_start && task.suggested_end && ' \u2013 '}
            {task.suggested_end && task.suggested_end.slice(0, 5)}
          </p>
        )}
      </div>
      <div className="flex gap-1 shrink-0">
        {task.status !== 'done' && (
          <Button size="sm" variant="ghost" onClick={() => onStatusChange(task.id, 'done')} title="Mark done">
            ✓
          </Button>
        )}
        {task.status !== 'skipped' && task.status !== 'done' && (
          <Button size="sm" variant="ghost" onClick={() => onStatusChange(task.id, 'skipped')} title="Skip">
            ⏭
          </Button>
        )}
        {(task.status === 'done' || task.status === 'skipped') && (
          <Button size="sm" variant="ghost" onClick={() => onStatusChange(task.id, 'pending')} title="Reset to pending">
            ↺
          </Button>
        )}
      </div>
    </div>
  );
}

export default function TodayPage() {
  const queryClient = useQueryClient();
  const [tasks, setTasks] = useState<Task[]>([]);

  const { data: plan, isLoading, error } = useQuery<TodayPlan>({
    queryKey: ['today-plan'],
    queryFn: async () => {
      const data = await getTodayPlan();
      setTasks(data.tasks || []);
      return data;
    },
  });

  const updateTaskMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: object }) => updateTask(id, data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['today-plan'] }),
    onError: () => toast.error('Failed to update task'),
  });

  const generateMutation = useMutation({
    mutationFn: generatePlan,
    onSuccess: () => {
      toast.success('Plan regenerated!');
      queryClient.invalidateQueries({ queryKey: ['today-plan'] });
    },
    onError: () => toast.error('Failed to regenerate plan'),
  });

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  );

  const handleStatusChange = (id: number, status: string) => {
    setTasks(prev => prev.map(t => t.id === id ? { ...t, status } : t));
    updateTaskMutation.mutate({ id, data: { status } });
  };

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (over && active.id !== over.id) {
      const oldIndex = tasks.findIndex(t => t.id === active.id);
      const newIndex = tasks.findIndex(t => t.id === over.id);
      const newTasks = arrayMove(tasks, oldIndex, newIndex);
      setTasks(newTasks);
      updateTaskMutation.mutate({ id: Number(active.id), data: { sort_order: newIndex } });
    }
  };

  if (isLoading) return <div className="flex justify-center py-12"><div className="text-muted-foreground">Loading today&apos;s plan...</div></div>;
  if (error) return (
    <div className="text-center py-12">
      <p className="text-muted-foreground mb-4">No plan found for today.</p>
      <Button onClick={() => generateMutation.mutate()} disabled={generateMutation.isPending}>
        {generateMutation.isPending ? 'Generating...' : 'Generate Plan'}
      </Button>
    </div>
  );

  return (
    <div className="max-w-2xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">Today&apos;s Plan</h1>
          {plan?.date && <p className="text-muted-foreground text-sm">{new Date(plan.date).toLocaleDateString('en-US', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' })}</p>}
        </div>
        <div className="flex gap-2">
          <Button asChild variant="outline" size="sm">
            <Link href="/events">Add Event</Link>
          </Button>
          <AddGoalDialog
            trigger={<Button variant="outline" size="sm">Add Goal</Button>}
          />
          <Button variant="outline" size="sm" onClick={() => generateMutation.mutate()} disabled={generateMutation.isPending}>
            {generateMutation.isPending ? 'Regenerating...' : 'Regenerate Plan'}
          </Button>
        </div>
      </div>

      {tasks.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center">
            <p className="text-muted-foreground mb-4">No tasks for today.</p>
            <Button onClick={() => generateMutation.mutate()} disabled={generateMutation.isPending}>
              Generate Plan
            </Button>
          </CardContent>
        </Card>
      ) : (
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
          <SortableContext items={tasks.map(t => t.id)} strategy={verticalListSortingStrategy}>
            <div className="space-y-2">
              {tasks.map(task => (
                <SortableTask key={task.id} task={task} onStatusChange={handleStatusChange} />
              ))}
            </div>
          </SortableContext>
        </DndContext>
      )}

      <div className="mt-6 text-sm text-muted-foreground text-center">
        {tasks.filter(t => t.status === 'done').length} / {tasks.length} tasks completed
      </div>
    </div>
  );
}
