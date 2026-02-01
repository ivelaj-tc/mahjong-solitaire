import type { ReactNode } from "react";

interface LoadingScreenProps {
  themeToggle: ReactNode;
  message: string;
}

export default function LoadingScreen({ themeToggle, message }: LoadingScreenProps) {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4">
      {themeToggle}
      <div className="h-12 w-12 animate-spin rounded-full border-4 border-[var(--gold)] border-t-[var(--jade)]" />
      <p className="text-lg text-[var(--muted)]">{message}</p>
    </div>
  );
}
