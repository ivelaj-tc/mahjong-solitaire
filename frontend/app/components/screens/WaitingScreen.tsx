import type { ReactNode } from "react";

interface WaitingScreenProps {
  themeToggle: ReactNode;
  statusMessage: string;
}

export default function WaitingScreen({ themeToggle, statusMessage }: WaitingScreenProps) {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-6">
      {themeToggle}
      <h1 className="font-display text-3xl font-bold text-[var(--jade)]">Waiting for Opponent</h1>
      <div className="h-12 w-12 animate-spin rounded-full border-4 border-[var(--gold)] border-t-[var(--jade)]" />
      <p className="text-[var(--muted)]">{statusMessage}</p>
    </div>
  );
}
