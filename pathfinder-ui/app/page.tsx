'use client';
import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { getGoals } from '@/lib/api';

interface Goal {
  id: number;
  title: string;
}

export default function HomePage() {
  const router = useRouter();
  useEffect(() => {
    getGoals()
      .then((goals: Goal[]) => {
        if (goals && goals.length > 0) {
          router.replace('/today');
        } else {
          router.replace('/onboarding');
        }
      })
      .catch(() => router.replace('/onboarding'));
  }, [router]);
  return (
    <div className="flex items-center justify-center min-h-[60vh]">
      <div className="text-muted-foreground">Loading...</div>
    </div>
  );
}
