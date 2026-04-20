'use client';
import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';
import { createGoal, updateUserProfile } from '@/lib/api';
import { toast } from 'sonner';

const GOAL_TAGS = ['career', 'health', 'education', 'personal', 'other'];

const primaryGoalSchema = z.object({
  title: z.string().min(2, 'Title must be at least 2 characters'),
  description: z.string().optional(),
  tag: z.string().min(1, 'Select a goal category'),
  dailyHours: z.number().min(0.5).max(24),
  preferredStartTime: z.string().optional(),
  timeline: z.string().optional(),
});

type PrimaryGoalForm = z.infer<typeof primaryGoalSchema>;

interface SecondaryGoal {
  title: string;
  description: string;
  tag: string;
}

export default function OnboardingPage() {
  const router = useRouter();
  const [step, setStep] = useState(1);
  const [selectedTag, setSelectedTag] = useState('');
  const [resumeFile, setResumeFile] = useState<File | null>(null);
  const [bio, setBio] = useState('');
  const [secondaryGoals, setSecondaryGoals] = useState<SecondaryGoal[]>([]);
  const [newSecondaryGoal, setNewSecondaryGoal] = useState<SecondaryGoal>({ title: '', description: '', tag: 'personal' });
  const [isSubmitting, setIsSubmitting] = useState(false);

  const { register, handleSubmit, formState: { errors }, setValue } = useForm<PrimaryGoalForm>({
    resolver: zodResolver(primaryGoalSchema),
    defaultValues: { dailyHours: 8, tag: '', preferredStartTime: '08:00' },
  });

  const totalSteps = 6;

  const handleTagSelect = (tag: string) => {
    setSelectedTag(tag);
    setValue('tag', tag);
  };

  const handleNextStep = () => { if (step < totalSteps) setStep(step + 1); };
  const handlePrevStep = () => { if (step > 1) setStep(step - 1); };

  const addSecondaryGoal = () => {
    if (newSecondaryGoal.title.trim()) {
      setSecondaryGoals([...secondaryGoals, { ...newSecondaryGoal }]);
      setNewSecondaryGoal({ title: '', description: '', tag: 'personal' });
    }
  };

  const removeSecondaryGoal = (index: number) => {
    setSecondaryGoals(secondaryGoals.filter((_, i) => i !== index));
  };

  const onSubmit = async (data: PrimaryGoalForm) => {
    setIsSubmitting(true);
    try {
      await createGoal({
        title: data.title,
        description: data.description || '',
        weight: 8,
        tags: [data.tag],
        timeline: data.timeline || '',
      });

      for (const sg of secondaryGoals) {
        await createGoal({
          title: sg.title,
          description: sg.description,
          weight: 5,
          tags: [sg.tag],
        });
      }

      // Save user profile (bio, resume, daily hours).
      const profileData = new FormData();
      if (bio) profileData.append('bio', bio);
      if (resumeFile) profileData.append('resume', resumeFile);
      profileData.append('daily_available_hours', String(data.dailyHours));
      await updateUserProfile(profileData).catch(() => {/* non-fatal */});

      toast.success('Goals created! Welcome to Pathfinder!');
      router.push('/today');
    } catch {
      toast.error('Failed to create goals. Please try again.');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="max-w-2xl mx-auto">
      <div className="mb-8">
        <h1 className="text-3xl font-bold mb-2">Welcome to Pathfinder</h1>
        <p className="text-muted-foreground">Let&apos;s set up your goals and schedule.</p>
        <div className="flex gap-2 mt-4">
          {Array.from({ length: totalSteps }).map((_, i) => (
            <div
              key={i}
              className={`h-2 flex-1 rounded-full transition-colors ${i < step ? 'bg-primary' : 'bg-muted'}`}
            />
          ))}
        </div>
        <p className="text-sm text-muted-foreground mt-2">Step {step} of {totalSteps}</p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)}>
        {/* Step 1: Goal Category */}
        {step === 1 && (
          <Card>
            <CardHeader>
              <CardTitle>What type of goal do you want to pursue?</CardTitle>
              <CardDescription>Choose the category that best describes your primary goal.</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
                {GOAL_TAGS.map((tag) => (
                  <button
                    key={tag}
                    type="button"
                    onClick={() => handleTagSelect(tag)}
                    className={`p-4 rounded-lg border-2 text-left capitalize font-medium transition-all ${
                      selectedTag === tag
                        ? 'border-primary bg-primary/10 text-primary'
                        : 'border-border hover:border-primary/50'
                    }`}
                  >
                    {tag}
                  </button>
                ))}
              </div>
              {errors.tag && <p className="text-destructive text-sm mt-2">{errors.tag.message}</p>}
              <div className="flex justify-end mt-6">
                <Button type="button" onClick={handleNextStep} disabled={!selectedTag}>
                  Next
                </Button>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Step 2: Title + Description */}
        {step === 2 && (
          <Card>
            <CardHeader>
              <CardTitle>Tell us about your goal</CardTitle>
              <CardDescription>Give your goal a clear title and description.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label htmlFor="title">Goal Title *</Label>
                <Input id="title" {...register('title')} placeholder="e.g., Get promoted to Senior Engineer" className="mt-1" />
                {errors.title && <p className="text-destructive text-sm mt-1">{errors.title.message}</p>}
              </div>
              <div>
                <Label htmlFor="description">Description (optional)</Label>
                <Textarea id="description" {...register('description')} placeholder="Describe what achieving this goal looks like..." className="mt-1" rows={4} />
              </div>
              <div>
                <Label htmlFor="timeline">Timeline (optional)</Label>
                <Input id="timeline" {...register('timeline')} placeholder="e.g. 3 months (leave empty for long-term)" className="mt-1" />
              </div>
              <div className="flex justify-between mt-6">
                <Button type="button" variant="outline" onClick={handlePrevStep}>Back</Button>
                <Button type="button" onClick={handleNextStep}>Next</Button>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Step 3: Timeline context */}
        {step === 3 && (
          <Card>
            <CardHeader>
              <CardTitle>Let&apos;s plan your first week</CardTitle>
              <CardDescription>We&apos;ll start with a one-week plan to get you moving. You can set longer-term milestones later.</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="rounded-lg border bg-primary/5 p-4 text-sm text-muted-foreground">
                📅 Your first plan will cover <strong className="text-foreground">7 days</strong>. After completing it, you can extend your timeline and add milestones.
              </div>
              <div className="flex justify-between mt-6">
                <Button type="button" variant="outline" onClick={handlePrevStep}>Back</Button>
                <Button type="button" onClick={handleNextStep}>Next</Button>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Step 4: Daily Hours */}
        {step === 4 && (
          <Card>
            <CardHeader>
              <CardTitle>Your daily schedule</CardTitle>
              <CardDescription>How much time can you dedicate each day?</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label htmlFor="dailyHours">Daily Available Hours (0.5–24)</Label>
                <Input
                  id="dailyHours"
                  type="number"
                  min={0.5}
                  max={24}
                  step={0.5}
                  {...register('dailyHours', { valueAsNumber: true })}
                  className="mt-1 w-32"
                />
                {errors.dailyHours && <p className="text-destructive text-sm mt-1">{errors.dailyHours.message}</p>}
              </div>
              <div>
                <Label htmlFor="preferredStartTime">Preferred Start Time</Label>
                <Input
                  id="preferredStartTime"
                  type="time"
                  {...register('preferredStartTime')}
                  className="mt-1 w-48"
                />
              </div>
              <div className="flex justify-between mt-6">
                <Button type="button" variant="outline" onClick={handlePrevStep}>Back</Button>
                <Button type="button" onClick={handleNextStep}>Next</Button>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Step 5: Secondary Goals */}
        {step === 5 && (
          <Card>
            <CardHeader>
              <CardTitle>Any secondary goals? (Optional)</CardTitle>
              <CardDescription>Add additional goals you want to track alongside your primary goal.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {secondaryGoals.map((sg, i) => (
                <div key={i} className="flex items-center gap-2 p-3 rounded-lg border bg-muted/50">
                  <div className="flex-1">
                    <p className="font-medium">{sg.title}</p>
                    {sg.description && <p className="text-sm text-muted-foreground">{sg.description}</p>}
                    <Badge variant="outline" className="mt-1 capitalize">{sg.tag}</Badge>
                  </div>
                  <Button type="button" variant="ghost" size="sm" onClick={() => removeSecondaryGoal(i)}>Remove</Button>
                </div>
              ))}
              <div className="border rounded-lg p-4 space-y-3">
                <p className="font-medium text-sm">Add a secondary goal</p>
                <Input
                  placeholder="Goal title"
                  value={newSecondaryGoal.title}
                  onChange={(e) => setNewSecondaryGoal({ ...newSecondaryGoal, title: e.target.value })}
                />
                <Textarea
                  placeholder="Description (optional)"
                  value={newSecondaryGoal.description}
                  onChange={(e) => setNewSecondaryGoal({ ...newSecondaryGoal, description: e.target.value })}
                  rows={2}
                />
                <select
                  className="w-full border rounded-md px-3 py-2 text-sm bg-background"
                  value={newSecondaryGoal.tag}
                  onChange={(e) => setNewSecondaryGoal({ ...newSecondaryGoal, tag: e.target.value })}
                >
                  {GOAL_TAGS.map(t => <option key={t} value={t} className="capitalize">{t}</option>)}
                </select>
                <Button type="button" variant="outline" onClick={addSecondaryGoal} disabled={!newSecondaryGoal.title.trim()}>
                  Add Secondary Goal
                </Button>
              </div>
              <div className="flex justify-between mt-6">
                <Button type="button" variant="outline" onClick={handlePrevStep}>Back</Button>
                <Button type="button" onClick={handleNextStep}>Next</Button>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Step 6: About You */}
        {step === 6 && (
          <Card>
            <CardHeader>
              <CardTitle>About You (Optional)</CardTitle>
              <CardDescription>Help Pathfinder understand your background to generate better plans.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label htmlFor="bio">Bio / Background</Label>
                <Textarea
                  id="bio"
                  placeholder="e.g., I'm a software engineer with 3 years of experience, looking to transition into machine learning..."
                  value={bio}
                  onChange={(e) => setBio(e.target.value)}
                  className="mt-1"
                  rows={5}
                />
              </div>
              <div>
                <Label htmlFor="resume">Upload Resume (optional)</Label>
                <Input
                  id="resume"
                  type="file"
                  accept=".pdf,.doc,.docx,.txt"
                  className="mt-1"
                  onChange={(e) => setResumeFile(e.target.files?.[0] || null)}
                />
                <p className="text-xs text-muted-foreground mt-1">Supported formats: PDF, DOC, DOCX, TXT</p>
              </div>
              <div className="flex justify-between mt-6">
                <Button type="button" variant="outline" onClick={handlePrevStep}>Back</Button>
                <Button type="submit" disabled={isSubmitting}>
                  {isSubmitting ? 'Creating...' : 'Get Started!'}
                </Button>
              </div>
            </CardContent>
          </Card>
        )}
      </form>
    </div>
  );
}
