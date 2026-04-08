'use client';
import { useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { authVerifyEmail } from '@/lib/api';

export default function VerifyEmailPage() {
  const searchParams = useSearchParams();
  const token = searchParams.get('token') || '';
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading');
  const [message, setMessage] = useState('');

  useEffect(() => {
    if (!token) {
      setStatus('error');
      setMessage('Invalid or missing activation link.');
      return;
    }
    authVerifyEmail(token)
      .then((data: { message?: string }) => {
        setStatus('success');
        setMessage(data.message || 'Email verified! You can now use all features.');
      })
      .catch((err: unknown) => {
        setStatus('error');
        const msg =
          (err as { response?: { data?: { message?: string } } })?.response?.data?.message ||
          'Verification failed. The link may have expired.';
        setMessage(msg);
      });
  }, [token]);

  return (
    <div className="max-w-md mx-auto mt-24">
      <Card>
        <CardHeader>
          <CardTitle>Email Verification</CardTitle>
          <CardDescription>
            {status === 'loading' && 'Verifying your email address…'}
            {status === 'success' && 'Your email has been verified.'}
            {status === 'error' && 'Verification failed.'}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {status === 'loading' && (
            <p className="text-muted-foreground text-sm">Please wait…</p>
          )}
          {status === 'success' && (
            <>
              <p className="text-sm">{message}</p>
              <Button asChild className="w-full">
                <Link href="/">Go to Pathfinder</Link>
              </Button>
            </>
          )}
          {status === 'error' && (
            <>
              <p className="text-destructive text-sm">{message}</p>
              <Button asChild variant="outline" className="w-full">
                <Link href="/login">Back to Sign In</Link>
              </Button>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
