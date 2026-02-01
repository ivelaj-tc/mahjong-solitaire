interface WinOverlayProps {
  show: boolean;
  showContent: boolean;
  winner: number;
  playerId: number | null;
  statusMessage: string;
  onReset: () => void;
}

export default function WinOverlay({
  show,
  showContent,
  winner,
  playerId,
  statusMessage,
  onReset,
}: WinOverlayProps) {
  if (!show) return null;

  return (
    <div className="win-overlay fixed inset-0 flex items-center justify-center">
      <div className="win-card rounded-2xl bg-[var(--win-mask)] p-8 text-center shadow-2xl">
        <div
          className={`win-card-inner transition-opacity duration-500 ${
            showContent ? "opacity-100 pointer-events-auto" : "opacity-0 pointer-events-none"
          }`}
        >
          <h2 className="font-display text-3xl font-bold text-[var(--jade)]">
            {winner === playerId ? "You Win!" : "You Lose!"}
          </h2>
          <p className="mt-2 text-[var(--muted)]">{statusMessage}</p>
          <button
            onClick={onReset}
            className="mt-6 rounded-lg bg-[var(--jade)] px-8 py-3 text-lg font-semibold text-white hover:bg-[var(--jade)]/90"
          >
            Play Again
          </button>
        </div>
      </div>
    </div>
  );
}
