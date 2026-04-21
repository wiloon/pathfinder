'use client';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getWeekAgenda, updateTask, deleteTask, type WeekAgenda, type DayAgenda, type AgendaTask, type AgendaGoal, type AgendaEvent } from '@/lib/api';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { toast } from 'sonner';

function formatDate(dateStr: string): string {
  const d = new Date(dateStr + 'T00:00:00');
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const diff = Math.round((d.getTime() - today.getTime()) / 86400000);
  const weekday = d.toLocaleDateString('zh-CN', { weekday: 'long' });
  const mmdd = d.toLocaleDateString('zh-CN', { month: 'long', day: 'numeric' });
  if (diff === 0) return `今天  ${mmdd} ${weekday}`;
  if (diff === 1) return `明天  ${mmdd} ${weekday}`;
  if (diff === -1) return `昨天  ${mmdd} ${weekday}`;
  return `${mmdd} ${weekday}`;
}

function isToday(dateStr: string): boolean {
  return dateStr === new Date().toISOString().slice(0, 10);
}

function isFuture(dateStr: string): boolean {
  return dateStr >= new Date().toISOString().slice(0, 10);
}

function TaskRow({ task, onStatusChange, onDelete }: { task: AgendaTask; onStatusChange: (id: number, status: string) => void; onDelete: (id: number) => void }) {
  const statusColors: Record<string, string> = {
    pending: 'bg-yellow-100 text-yellow-800',
    done: 'bg-green-100 text-green-800',
    skipped: 'bg-gray-100 text-gray-500',
  };
  return (
    <div className={`flex items-start gap-3 py-2 px-3 rounded-md hover:bg-muted/50 transition-colors ${task.status === 'done' ? 'opacity-60' : ''}`}>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 flex-wrap">
          <span className={`text-sm font-medium ${task.status === 'done' ? 'line-through text-muted-foreground' : ''}`}>
            {task.title}
          </span>
          <Badge className={`text-xs ${statusColors[task.status] || 'bg-gray-100'}`} variant="outline">
            {task.status}
          </Badge>
        </div>
        {task.description && <p className="text-xs text-muted-foreground mt-0.5">{task.description}</p>}
        {(task.suggested_start || task.suggested_end) && (
          <p className="text-xs text-muted-foreground mt-0.5">
            🕐 {task.suggested_start?.slice(0, 5)}{task.suggested_start && task.suggested_end && ' – '}{task.suggested_end?.slice(0, 5)}
          </p>
        )}
      </div>
      <div className="flex gap-1 shrink-0 mt-0.5">
        {task.status !== 'done' && (
          <Button size="sm" variant="ghost" className="h-6 w-6 p-0" onClick={() => onStatusChange(task.id, 'done')} title="完成">
            ✓
          </Button>
        )}
        {task.status !== 'skipped' && task.status !== 'done' && (
          <Button size="sm" variant="ghost" className="h-6 w-6 p-0" onClick={() => onStatusChange(task.id, 'skipped')} title="跳过">
            ⏭
          </Button>
        )}
        {(task.status === 'done' || task.status === 'skipped') && (
          <Button size="sm" variant="ghost" className="h-6 w-6 p-0" onClick={() => onStatusChange(task.id, 'pending')} title="重置">
            ↺
          </Button>
        )}
        <Button size="sm" variant="ghost" className="h-6 w-6 p-0 text-muted-foreground hover:text-destructive" onClick={() => onDelete(task.id)} title="删除">
          ✕
        </Button>
      </div>
    </div>
  );
}

function GoalRow({ goal }: { goal: AgendaGoal }) {
  let tags: string[] = [];
  try { tags = JSON.parse(goal.tags); } catch { /* ignore */ }
  return (
    <div className="flex items-start gap-3 py-2 px-3 rounded-md hover:bg-muted/50 transition-colors">
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-sm font-medium">🎯 {goal.title}</span>
          <Badge className="text-xs bg-blue-50 text-blue-700" variant="outline">P{goal.weight}</Badge>
          {goal.timeline && (
            <Badge className="text-xs bg-purple-50 text-purple-700" variant="outline">
              {goal.timeline}
            </Badge>
          )}
        </div>
        {goal.description && <p className="text-xs text-muted-foreground mt-0.5">{goal.description}</p>}
        {tags.length > 0 && (
          <div className="flex gap-1 mt-1 flex-wrap">
            {tags.map((t, i) => (
              <span key={i} className="text-xs bg-muted px-1.5 py-0.5 rounded">{t}</span>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function EventRow({ event }: { event: AgendaEvent }) {
  return (
    <div className="flex items-start gap-3 py-2 px-3 rounded-md hover:bg-muted/50 transition-colors">
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-sm font-medium">📅 {event.title}</span>
          <Badge
            className={`text-xs ${event.status === 'completed' ? 'bg-green-50 text-green-700' : 'bg-orange-50 text-orange-700'}`}
            variant="outline"
          >
            {event.status === 'completed' ? '已完成' : '待办'}
          </Badge>
        </div>
        {event.description && <p className="text-xs text-muted-foreground mt-0.5">{event.description}</p>}
      </div>
    </div>
  );
}

function DaySection({ day, onTaskStatusChange, onTaskDelete }: { day: DayAgenda; onTaskStatusChange: (id: number, status: string) => void; onTaskDelete: (id: number) => void }) {
  const total = day.tasks.length + day.goals.length + day.events.length;
  const today = isToday(day.date);
  const future = isFuture(day.date);
  const empty = total === 0;

  return (
    <div className={`border rounded-lg overflow-hidden ${today ? 'border-primary/50 shadow-sm' : 'border-border'}`}>
      <div className={`px-4 py-2 flex items-center justify-between ${today ? 'bg-primary/5' : 'bg-muted/30'}`}>
        <span className={`font-semibold text-sm ${today ? 'text-primary' : future ? 'text-foreground' : 'text-muted-foreground'}`}>
          {formatDate(day.date)}
        </span>
        {total > 0 && (
          <span className="text-xs text-muted-foreground">
            {day.tasks.filter(t => t.status === 'done').length}/{day.tasks.length} 任务
            {day.goals.length > 0 && `  ·  ${day.goals.length} 目标`}
            {day.events.length > 0 && `  ·  ${day.events.length} 事件`}
          </span>
        )}
      </div>
      {empty ? (
        <div className="px-4 py-3 text-xs text-muted-foreground">暂无安排</div>
      ) : (
        <div className="divide-y divide-border/50">
          {day.tasks.length > 0 && (
            <div className="py-1">
              {day.tasks.map(task => (
                <TaskRow key={task.id} task={task} onStatusChange={onTaskStatusChange} onDelete={onTaskDelete} />
              ))}
            </div>
          )}
          {day.goals.length > 0 && (
            <div className="py-1">
              {day.goals.map(goal => (
                <GoalRow key={goal.id} goal={goal} />
              ))}
            </div>
          )}
          {day.events.length > 0 && (
            <div className="py-1">
              {day.events.map(event => (
                <EventRow key={event.id} event={event} />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export function WeekAgendaView() {
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery<WeekAgenda>({
    queryKey: ['week-agenda'],
    queryFn: () => getWeekAgenda(),
  });

  const updateTaskMutation = useMutation({
    mutationFn: ({ id, status }: { id: number; status: string }) => updateTask(id, { status }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['week-agenda'] }),
    onError: () => toast.error('更新失败'),
  });

  const deleteTaskMutation = useMutation({
    mutationFn: (id: number) => deleteTask(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['week-agenda'] }),
    onError: () => toast.error('删除失败'),
  });

  if (isLoading) return <div className="py-12 text-center text-muted-foreground text-sm">加载中...</div>;
  if (error || !data) return <div className="py-12 text-center text-muted-foreground text-sm">加载失败</div>;

  const handleTaskStatusChange = (id: number, status: string) => {
    updateTaskMutation.mutate({ id, status });
  };

  const handleTaskDelete = (id: number) => {
    deleteTaskMutation.mutate(id);
  };

  return (
    <div className="space-y-3">
      {data.days.filter(day => (day.tasks?.length ?? 0) + (day.goals?.length ?? 0) + (day.events?.length ?? 0) > 0).map(day => (
        <DaySection key={day.date} day={day} onTaskStatusChange={handleTaskStatusChange} onTaskDelete={handleTaskDelete} />
      ))}

      {data.unscheduled.length > 0 && (
        <div className="border rounded-lg overflow-hidden border-dashed border-border">
          <div className="px-4 py-2 bg-muted/20">
            <span className="font-semibold text-sm text-muted-foreground">待规划目标</span>
          </div>
          <div className="py-1">
            {data.unscheduled.map(goal => (
              <GoalRow key={goal.id} goal={goal} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
