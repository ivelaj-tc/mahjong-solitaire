import type { ReactNode } from "react";

interface JoinScreenProps {
  themeToggle: ReactNode;
  nameInput: string;
  onNameChange: (value: string) => void;
  onJoin: () => void;
  connected: boolean;
  playWithBot: boolean;
  onPlayWithBotChange: (checked: boolean) => void;
}

export default function JoinScreen({
  themeToggle,
  nameInput,
  onNameChange,
  onJoin,
  connected,
  playWithBot,
  onPlayWithBotChange,
}: JoinScreenProps) {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-8 p-8">
      {themeToggle}
      <h1 className="font-display text-4xl font-bold tracking-wide text-[var(--jade)]">
        Mahjong Push Arena
      </h1>
      <div className="flex flex-col items-center gap-4">
        <input
          type="text"
          value={nameInput}
          onChange={(event) => onNameChange(event.target.value)}
          onKeyDown={(event) => event.key === "Enter" && onJoin()}
          placeholder="Enter your name"
          className="w-64 rounded-lg border-2 border-[var(--gold)] bg-[var(--surface-soft)] px-4 py-3 text-center text-lg text-[var(--foreground)] outline-none focus:border-[var(--jade)] transition-colors"
        />
        <label className="flex items-center gap-3 text-sm text-[var(--muted)]">
          <input
            type="checkbox"
            checked={playWithBot}
            onChange={(event) => onPlayWithBotChange(event.target.checked)}
            className="h-4 w-4 rounded border-[var(--gold)] text-[var(--jade)] focus:ring-[var(--jade)]"
          />
          Play with bot
        </label>
        <button
          onClick={onJoin}
          disabled={!nameInput.trim() || !connected}
          className="rounded-lg bg-[var(--jade)] px-8 py-3 text-lg font-semibold text-white transition-all hover:bg-[var(--jade)]/90 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {connected ? "Join Game" : "Connecting..."}
        </button>
      </div>
      <p className="text-sm text-[var(--muted)]">
        {connected ? "Connected to server" : "Connecting to server..."}
      </p>
    </div>
  );
}
