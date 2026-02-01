import type { ReactNode } from "react";

const rpsIcons: Record<string, string> = {
  rock: "🪨",
  paper: "📄",
  scissors: "✂️",
};

interface RpsRevealScreenProps {
  themeToggle: ReactNode;
  opponentName?: string | null;
  myChoice?: string | null;
  opponentChoice?: string | null;
}

export default function RpsRevealScreen({
  themeToggle,
  opponentName,
  myChoice,
  opponentChoice,
}: RpsRevealScreenProps) {
  return (
    <div className="rps-reveal-screen flex min-h-screen flex-col items-center justify-center gap-8 p-8">
      {themeToggle}
      <h1 className="font-display text-3xl font-bold text-[var(--jade)]">RPS Reveal</h1>
      <p className="text-lg text-[var(--muted)]">Shuffling tiles...</p>
      <div className="flex flex-col items-center gap-6 md:flex-row">
        <div className="rps-flip-card rps-flip-left h-52 w-56">
          <div className="rps-flip-inner">
            <div className="rps-flip-face rps-flip-front border-2 border-[var(--jade)]/70 bg-[var(--surface)] shadow-lg">
              <p className="text-xs uppercase tracking-[0.25em] text-[var(--muted)]">You</p>
              <div className="mt-3 text-6xl">{rpsIcons[myChoice ?? ""] ?? "❓"}</div>
              <p className="mt-2 text-lg font-semibold capitalize text-[var(--foreground)]">
                {myChoice ?? "Waiting"}
              </p>
            </div>
            <div className="rps-flip-face rps-flip-back border-2 border-[var(--gold)]/40 bg-[var(--surface-soft)] text-[var(--muted)] shadow-lg">
              <span className="text-3xl uppercase tracking-[0.2em]">RPS</span>
            </div>
          </div>
        </div>
        <div className="rps-flip-card rps-flip-right h-52 w-56">
          <div className="rps-flip-inner">
            <div className="rps-flip-face rps-flip-front border-2 border-[var(--gold)]/70 bg-[var(--surface)] shadow-lg">
              <p className="text-xs uppercase tracking-[0.25em] text-[var(--muted)]">
                {opponentName ?? "Opponent"}
              </p>
              <div className="mt-3 text-6xl">{rpsIcons[opponentChoice ?? ""] ?? "❓"}</div>
              <p className="mt-2 text-lg font-semibold capitalize text-[var(--foreground)]">
                {opponentChoice ?? "Waiting"}
              </p>
            </div>
            <div className="rps-flip-face rps-flip-back border-2 border-[var(--jade)]/40 bg-[var(--surface-soft)] text-[var(--muted)] shadow-lg">
              <span className="text-3xl uppercase tracking-[0.2em]">RPS</span>
            </div>
          </div>
        </div>
      </div>
      <p className="text-sm text-[var(--muted)]">Starting the match...</p>
    </div>
  );
}
