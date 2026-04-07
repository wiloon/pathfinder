'use client';
import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { getEvents, createEvent, deleteEvent, submitEventRetro } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { toast } from 'sonner';

const eventSchema = z.object({
  title: z.string().min(2),
  description: z.string().optional(),
  event_date: z.string().min(1, 'Date is required'),
});
type EventForm = z.infer<typeof eventSchema>;

const retroSchema = z.object({
  notes: z.string().min(1, 'Please add some notes'),
  outcome: z.string().optional(),
});
type RetroForm = z.infer<typeof retroSchema>;

interface Event {
  id: number;
  title: string;
  description?: string;
  event_date: string;
  status: string;
}

const statusColors: Record<string, string> = {
  upcoming: 'bg-blue-100 text-blue-800',
  completed: 'bg-green-100 text-green-800',
  cancelled: 'bg-red-100 text-red-800',
};

export default function EventsPage() {
  const queryClient = useQueryClient();
  const [addOpen, setAddOpen] = useState(false);
  const [retroEvent, setRetroEvent] = useState<Event | null>(null);
  const [retroOpen, setRetroOpen] = useState(false);
  const [imageFile, setImageFile] = useState<File | null>(null);

  const { data: events = [], isLoading } = useQuery<Event[]>({
    queryKey: ['events'],
    queryFn: getEvents,
  });

  const { register, handleSubmit, reset, formState: { errors } } = useForm<EventForm>({ resolver: zodResolver(eventSchema) });
  const { register: regRetro, handleSubmit: handleRetroSubmit, reset: resetRetro } = useForm<RetroForm>({ resolver: zodResolver(retroSchema) });

  const createMutation = useMutation({
    mutationFn: (data: EventForm) => {
      if (imageFile) {
        const fd = new FormData();
        fd.append('title', data.title);
        if (data.description) fd.append('description', data.description);
        fd.append('event_date', data.event_date);
        fd.append('image', imageFile);
        return createEvent(fd);
      }
      return createEvent(data);
    },
    onSuccess: () => { toast.success('Event created!'); queryClient.invalidateQueries({ queryKey: ['events'] }); setAddOpen(false); reset(); setImageFile(null); },
    onError: () => toast.error('Failed to create event'),
  });

  const deleteMutation = useMutation({
    mutationFn: deleteEvent,
    onSuccess: () => { toast.success('Event deleted'); queryClient.invalidateQueries({ queryKey: ['events'] }); },
    onError: () => toast.error('Failed to delete event'),
  });

  const retroMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: RetroForm }) => submitEventRetro(id, data),
    onSuccess: () => { toast.success('Retro submitted!'); queryClient.invalidateQueries({ queryKey: ['events'] }); setRetroOpen(false); resetRetro(); },
    onError: () => toast.error('Failed to submit retro'),
  });

  if (isLoading) return <div className="flex justify-center py-12"><div className="text-muted-foreground">Loading events...</div></div>;

  return (
    <div className="max-w-3xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold">Events</h1>
        <Dialog open={addOpen} onOpenChange={setAddOpen}>
          <DialogTrigger asChild>
            <Button>Add Event</Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader><DialogTitle>Add New Event</DialogTitle></DialogHeader>
            <form onSubmit={handleSubmit((data) => createMutation.mutate(data))} className="space-y-4">
              <div>
                <Label>Title *</Label>
                <Input {...register('title')} placeholder="Event title" className="mt-1" />
                {errors.title && <p className="text-destructive text-sm mt-1">{errors.title.message}</p>}
              </div>
              <div>
                <Label>Description</Label>
                <Textarea {...register('description')} placeholder="Description" className="mt-1" rows={3} />
              </div>
              <div>
                <Label>Date *</Label>
                <Input type="datetime-local" {...register('event_date')} className="mt-1" />
                {errors.event_date && <p className="text-destructive text-sm mt-1">{errors.event_date.message}</p>}
              </div>
              <div>
                <Label>Image (optional)</Label>
                <Input type="file" accept="image/*" className="mt-1" onChange={(e) => setImageFile(e.target.files?.[0] || null)} />
              </div>
              <Button type="submit" className="w-full" disabled={createMutation.isPending}>
                {createMutation.isPending ? 'Creating...' : 'Create Event'}
              </Button>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      {events.length === 0 ? (
        <Card>
          <CardContent className="py-12 text-center">
            <p className="text-muted-foreground">No events yet. Add an upcoming event!</p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {events.map((event) => (
            <Card key={event.id}>
              <CardContent className="pt-4 pb-4">
                <div className="flex items-start justify-between gap-3">
                  <div className="flex-1">
                    <div className="flex items-center gap-2 flex-wrap">
                      <h3 className="font-semibold">{event.title}</h3>
                      <Badge className={statusColors[event.status] || 'bg-gray-100'} variant="outline">
                        {event.status}
                      </Badge>
                    </div>
                    {event.description && <p className="text-sm text-muted-foreground mt-1">{event.description}</p>}
                    <p className="text-xs text-muted-foreground mt-1">
                      {new Date(event.event_date).toLocaleString('en-US', { dateStyle: 'medium', timeStyle: 'short' })}
                    </p>
                  </div>
                  <div className="flex gap-2 shrink-0">
                    {event.status === 'completed' && (
                      <Button size="sm" variant="outline" onClick={() => { setRetroEvent(event); setRetroOpen(true); }}>
                        Add Retro
                      </Button>
                    )}
                    <Button size="sm" variant="destructive" onClick={() => deleteMutation.mutate(event.id)}>Delete</Button>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Retro Dialog */}
      <Dialog open={retroOpen} onOpenChange={setRetroOpen}>
        <DialogContent>
          <DialogHeader><DialogTitle>Event Retrospective: {retroEvent?.title}</DialogTitle></DialogHeader>
          <form onSubmit={handleRetroSubmit((data) => retroEvent && retroMutation.mutate({ id: retroEvent.id, data }))} className="space-y-4">
            <div>
              <Label>Notes *</Label>
              <Textarea {...regRetro('notes')} placeholder="How did it go?" className="mt-1" rows={4} />
            </div>
            <div>
              <Label>Outcome</Label>
              <Input {...regRetro('outcome')} placeholder="e.g., success, partial, missed" className="mt-1" />
            </div>
            <Button type="submit" className="w-full" disabled={retroMutation.isPending}>
              {retroMutation.isPending ? 'Submitting...' : 'Submit Retro'}
            </Button>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
