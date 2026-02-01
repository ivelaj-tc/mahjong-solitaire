import type { ReactNode } from "react";

interface RpsScreenProps {
  themeToggle: ReactNode;
  statusMessage: string;
  rpsChoice: string | null;
  hasChosen: boolean;
  onChoose: (choice: string) => void;
}

export default function RpsScreen({
  themeToggle,
  statusMessage,
  rpsChoice,
  hasChosen,
  onChoose,
}: RpsScreenProps) {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-8 p-8">
      {themeToggle}
      <h1 className="font-display text-3xl font-bold text-[var(--jade)]">Rock Paper Scissors</h1>
      <p className="text-lg text-[var(--muted)]">{statusMessage}</p>
      <div className="flex gap-4">
        {["rock", "paper", "scissors"].map((choice) => (
          <button
            key={choice}
            onClick={() => onChoose(choice)}
            disabled={hasChosen}
            className={`flex h-24 w-24 flex-col items-center justify-center rounded-xl border-3 transition-all ${
              rpsChoice === choice
                ? "border-[var(--jade)] bg-[var(--jade)]/10"
                : "border-[var(--gold)] bg-[var(--surface-soft)] hover:border-[var(--jade)] hover:bg-[var(--jade)]/5"
            } ${hasChosen && rpsChoice !== choice ? "opacity-50" : ""}`}
          >
            <span className="text-4xl">
              {choice === "rock" ? "🪨" : choice === "paper" ? "📄" : "✂️"}
            </span>
            <span className="mt-1 text-sm font-medium capitalize text-[var(--foreground)]">{choice}</span>
          </button>
        ))}
      </div>
      {hasChosen && <p className="text-[var(--muted)]">Waiting for opponent to choose...</p>}
    </div>
  );
}
