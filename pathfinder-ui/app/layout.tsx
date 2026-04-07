import type { Metadata } from 'next';
import './globals.css';
import { Providers } from './providers';
import Link from 'next/link';

export const metadata: Metadata = {
  title: 'Pathfinder',
  description: 'Your personal goal and productivity tracker',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <Providers>
          <header className="border-b bg-background">
            <nav className="container mx-auto px-4 h-14 flex items-center gap-6">
              <Link href="/" className="font-bold text-lg text-primary">Pathfinder</Link>
              <div className="flex items-center gap-4 ml-4">
                <Link href="/today" className="text-sm font-medium hover:text-primary transition-colors">Today</Link>
                <Link href="/goals" className="text-sm font-medium hover:text-primary transition-colors">Goals</Link>
                <Link href="/events" className="text-sm font-medium hover:text-primary transition-colors">Events</Link>
                <Link href="/checkin" className="text-sm font-medium hover:text-primary transition-colors">Check-In</Link>
              </div>
            </nav>
          </header>
          <main className="container mx-auto px-4 py-8">
            {children}
          </main>
        </Providers>
      </body>
    </html>
  );
}
