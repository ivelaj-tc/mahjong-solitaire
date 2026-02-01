import type { ReactNode } from "react";

import type { GameState, Player, Tile } from "../../types";
import PlayerBoard from "./PlayerBoard";
import TileImage from "./TileImage";
import WinOverlay from "./WinOverlay";

interface GameScreenProps {
  themeToggle: ReactNode;
  gameState: GameState;
  currentPlayer?: Player;
  opponent?: Player;
  isMyTurn: boolean;
  playerId: number | null;
  showWinOverlay: boolean;
  showWinContent: boolean;
  onPush: (column: number) => void;
  onReset: () => void;
}

export default function GameScreen({
  themeToggle,
  gameState,
  currentPlayer,
  opponent,
  isMyTurn,
  playerId,
  showWinOverlay,
  showWinContent,
  onPush,
  onReset,
}: GameScreenProps) {
  const sharedTile = gameState.sharedTile;

  return (
    <div className="flex min-h-screen flex-col items-center gap-6 p-6">
      {themeToggle}
      <header className="flex w-full max-w-4xl items-center justify-between">
        <h1 className="font-display text-2xl font-bold text-[var(--jade)]">Mahjong Push Arena</h1>
        <div className="flex items-center gap-4">
          <span className="text-sm text-[var(--muted)]">Tiles left: {gameState.remainingTiles}</span>
          {gameState.phase === "gameover" && (
            <button
              onClick={onReset}
              className="rounded-lg bg-[var(--jade)] px-4 py-2 text-sm font-semibold text-white hover:bg-[var(--jade)]/90"
            >
              Rematch
            </button>
          )}
        </div>
      </header>

      <div className="rounded-lg bg-[var(--surface)] px-6 py-3 text-center shadow-sm">
        <p className="text-lg font-medium text-[var(--foreground)]">{gameState.statusMessage}</p>
        {gameState.phase === "playing" && (
          <p className={`text-sm ${isMyTurn ? "text-[var(--jade)] font-semibold" : "text-[var(--muted)]"}`}>
            {isMyTurn ? "Your turn!" : "Opponent's turn"}
          </p>
        )}
      </div>

      {sharedTile && gameState.phase === "playing" && (
        <div className="flex flex-col items-center gap-2">
          <p className="text-sm font-medium text-[var(--muted)]">Current Tile</p>
          <div className="float-slow glow-sweep rounded-xl border-3 border-[var(--gold)] bg-[var(--panel)] p-2 shadow-lg">
            <TileImage
              symbol={sharedTile.symbol}
              alt={sharedTile.symbol}
              width={80}
              height={100}
              className="rounded-lg"
              fallback={
                <div className="flex h-[100px] w-[80px] items-center justify-center rounded-lg bg-[var(--tile-empty)]">
                  <span className="text-2xl">{sharedTile.symbol}</span>
                </div>
              }
            />
          </div>
          <span
            className={`rounded-full px-3 py-1 text-xs font-medium ${
              sharedTile.category === "animals"
                ? "bg-[var(--jade)]/20 text-[var(--jade)]"
                : sharedTile.category === "foods"
                ? "bg-[var(--crimson)]/20 text-[var(--crimson)]"
                : "bg-[var(--muted)]/20 text-[var(--muted)]"
            }`}
          >
            {sharedTile.category}
          </span>
        </div>
      )}

      <div className="flex w-full max-w-4xl flex-col gap-8 lg:flex-row lg:justify-center lg:gap-12">
        {currentPlayer && (
          <PlayerBoard
            player={currentPlayer}
            isCurrentPlayer={true}
            isMyTurn={isMyTurn}
            gamePhase={gameState.phase}
            sharedTile={gameState.sharedTile}
            onPush={onPush}
          />
        )}
        {opponent && (
          <PlayerBoard
            player={opponent}
            isCurrentPlayer={false}
            isMyTurn={false}
            gamePhase={gameState.phase}
            sharedTile={gameState.sharedTile}
            onPush={() => {}}
          />
        )}
      </div>

      <WinOverlay
        show={showWinOverlay && gameState.phase === "gameover"}
        showContent={showWinContent}
        winner={gameState.winner}
        playerId={playerId}
        statusMessage={gameState.statusMessage}
        onReset={onReset}
      />
    </div>
  );
}
