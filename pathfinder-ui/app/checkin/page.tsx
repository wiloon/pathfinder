'use client';
import { useQuery, useMutation } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { getTodayCheckin, submitCheckin } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { toast } from 'sonner';
import Link from 'next/link';
import { useState } from 'react';

const checkinSchema = z.object({
  completed: z.string().min(1, 'Please describe what you accomplished'),
  blocked: z.string().optional(),
  tomorrow_focus: z.string().optional(),
});

type CheckinForm = z.infer<typeof checkinSchema>;

interface Checkin {
  id: number;
  completed: string;
  blocked?: string;
  tomorrow_focus?: string;
  created_at: string;
}

export default function CheckinPage() {
  const [submitted, setSubmitted] = useState(false);

  const { data: existingCheckin, isLoading } = useQuery<Checkin | null>({
    queryKey: ['today-checkin'],
    queryFn: async () => {
      try {
        return await getTodayCheckin();
      } catch {
        return null;
      }
    },
  });

  const { register, handleSubmit, formState: { errors } } = useForm<CheckinForm>({
    resolver: zodResolver(checkinSchema),
  });

  const mutation = useMutation({
    mutationFn: submitCheckin,
    onSuccess: () => {
      toast.success('Check-in submitted!');
      setSubmitted(true);
    },
    onError: () => toast.error('Failed to submit check-in'),
  });

  if (isLoading) return <div className="flex justify-center py-12"><div className="text-muted-foreground">Loading...</div></div>;

  const checkin = existingCheckin;

  if ((checkin && checkin.id) || submitted) {
    return (
      <div className="max-w-xl mx-auto">
        <Card>
          <CardHeader>
            <CardTitle className="text-green-600">✓ Check-in Complete</CardTitle>
            <CardDescription>You&apos;ve already checked in for today.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {checkin && (
              <>
                <div>
                  <p className="font-medium text-sm text-muted-foreground">What you accomplished:</p>
                  <p className="mt-1">{checkin.completed}</p>
                </div>
                {checkin.blocked && (
                  <div>
                    <p className="font-medium text-sm text-muted-foreground">Blockers:</p>
                    <p className="mt-1">{checkin.blocked}</p>
                  </div>
                )}
                {checkin.tomorrow_focus && (
                  <div>
                    <p className="font-medium text-sm text-muted-foreground">Tomorrow&apos;s focus:</p>
                    <p className="mt-1">{checkin.tomorrow_focus}</p>
                  </div>
                )}
              </>
            )}
            <Button asChild className="w-full mt-4">
              <Link href="/today">Back to Today</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="max-w-xl mx-auto">
      <h1 className="text-2xl font-bold mb-2">Daily Check-In</h1>
      <p className="text-muted-foreground mb-6">Reflect on your day and prepare for tomorrow.</p>
      <form onSubmit={handleSubmit((data) => mutation.mutate(data))}>
        <Card>
          <CardContent className="pt-6 space-y-5">
            <div>
              <Label htmlFor="completed">What did you accomplish today? *</Label>
              <Textarea id="completed" {...register('completed')} placeholder="Describe your accomplishments..." className="mt-1" rows={4} />
              {errors.completed && <p className="text-destructive text-sm mt-1">{errors.completed.message}</p>}
            </div>
            <div>
              <Label htmlFor="blocked">What blocked you or was harder than expected?</Label>
              <Textarea id="blocked" {...register('blocked')} placeholder="Any challenges or blockers..." className="mt-1" rows={3} />
            </div>
            <div>
              <Label htmlFor="tomorrow_focus">What&apos;s most important for tomorrow?</Label>
              <Textarea id="tomorrow_focus" {...register('tomorrow_focus')} placeholder="Top priorities for tomorrow..." className="mt-1" rows={3} />
            </div>
            <Button type="submit" className="w-full" disabled={mutation.isPending}>
              {mutation.isPending ? 'Submitting...' : 'Submit Check-In'}
            </Button>
          </CardContent>
        </Card>
      </form>
    </div>
  );
}
