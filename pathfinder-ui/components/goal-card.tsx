'use client';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';

export interface Goal {
  id: number;
  title: string;
  description?: string;
  weight: number;
  tags: string; // JSON array string from API
  status: string;
  timeline?: string;
}

export function parseTags(tags: string): string[] {
  try {
    const parsed = JSON.parse(tags);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

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
  selected?: boolean;
  onSelect?: (id: number) => void;
}) {
  const tags = parseTags(goal.tags);
  return (
    <Card
      className={`${selected ? 'ring-2 ring-primary' : ''} ${onSelect ? 'cursor-pointer' : ''}`}
      onClick={onSelect ? () => onSelect(goal.id) : undefined}
    >
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
          <Button size="sm" variant="outline" onClick={(e) => { e.stopPropagation(); onEdit(goal); }}>Edit</Button>
          <Button size="sm" variant="destructive" onClick={(e) => { e.stopPropagation(); onDelete(goal.id); }}>Delete</Button>
        </div>
      </CardContent>
    </Card>
  );
}
