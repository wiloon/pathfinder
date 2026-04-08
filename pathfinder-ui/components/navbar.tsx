'use client';
import Link from 'next/link';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import { authGetMe, authLogout } from '@/lib/api';
import { Button } from '@/components/ui/button';

interface Me {
  user_id: number;
  username: string;
  status: string;
}

export function Navbar() {
  const router = useRouter();
  const queryClient = useQueryClient();

  const { data: me } = useQuery<Me | null>({
    queryKey: ['me'],
    queryFn: authGetMe,
    retry: false,
    staleTime: 1000 * 60 * 5,
  });

  const handleLogout = async () => {
    await authLogout();
    queryClient.clear();
    router.push('/login');
  };

  return (
    <header className="border-b bg-background">
      <nav className="container mx-auto px-4 h-14 flex items-center gap-6">
        <Link href="/" className="font-bold text-lg text-primary">Pathfinder</Link>
        {me && (
          <div className="flex items-center gap-4 ml-4">
            <Link href="/today" className="text-sm font-medium hover:text-primary transition-colors">Today</Link>
            <Link href="/goals" className="text-sm font-medium hover:text-primary transition-colors">Goals</Link>
            <Link href="/events" className="text-sm font-medium hover:text-primary transition-colors">Events</Link>
            <Link href="/checkin" className="text-sm font-medium hover:text-primary transition-colors">Check-In</Link>
          </div>
        )}
        <div className="ml-auto flex items-center gap-3">
          {me ? (
            <>
              <span className="text-sm font-medium">{me.username}</span>
              <Button variant="outline" size="sm" onClick={handleLogout}>
                Sign Out
              </Button>
            </>
          ) : (
            <>
              <Link href="/login" className="text-sm font-medium hover:text-primary transition-colors">Sign In</Link>
              <Link
                href="/register"
                className="text-sm font-medium bg-primary text-primary-foreground px-3 py-1.5 rounded-md hover:opacity-90 transition-opacity"
              >
                Sign Up
              </Link>
            </>
          )}
        </div>
      </nav>
    </header>
  );
}
