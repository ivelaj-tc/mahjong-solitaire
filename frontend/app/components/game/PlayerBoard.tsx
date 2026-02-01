import type { GamePhase, Player, Tile } from "../../types";
import TileImage from "./TileImage";

interface PlayerBoardProps {
  player: Player;
  isCurrentPlayer: boolean;
  isMyTurn: boolean;
  gamePhase: GamePhase;
  sharedTile: Tile | null;
  onPush: (column: number) => void;
}

export default function PlayerBoard({
  player,
  isCurrentPlayer,
  isMyTurn,
  gamePhase,
  sharedTile,
  onPush,
}: PlayerBoardProps) {
  const canPush = isCurrentPlayer && isMyTurn && gamePhase === "playing" && sharedTile;
  const canPushColumn = (column: number) => {
    if (!canPush || !sharedTile) return false;
    if (sharedTile.category !== "blank" && sharedTile.category !== player.category) return false;
    if (sharedTile.category === "blank") return true;
    let hasMatchingFaceup = false;
    for (let col = 0; col < player.board[0].length; col += 1) {
      let matchAnchor: Tile | null = null;
      for (let row = player.board.length - 1; row >= 0; row -= 1) {
        const cell = player.board[row][col];
        if (cell.id !== 0) {
          matchAnchor = cell;
          break;
        }
      }
      if (matchAnchor && (matchAnchor.category === "blank" || matchAnchor.symbol === sharedTile.symbol)) {
        hasMatchingFaceup = true;
        break;
      }
    }
    let anchor: Tile | null = null;
    for (let row = player.board.length - 1; row >= 0; row -= 1) {
      const cell = player.board[row][column];
      if (cell.id !== 0) {
        anchor = cell;
        break;
      }
    }
    if (!anchor) return !hasMatchingFaceup;
    if (anchor.category === "blank") return true;
    return anchor.symbol === sharedTile.symbol;
  };

  return (
    <div className="flex flex-col items-center gap-3">
      <div className="flex items-center gap-2">
        <h3 className={`text-lg font-semibold ${isCurrentPlayer ? "text-[var(--jade)]" : "text-[var(--muted)]"}`}>
          {player.name} {isCurrentPlayer && "(You)"}
        </h3>
        <span
          className={`rounded-full px-2 py-0.5 text-xs font-medium ${
            player.category === "animals"
              ? "bg-[var(--jade)]/20 text-[var(--jade)]"
              : player.category === "foods"
              ? "bg-[var(--crimson)]/20 text-[var(--crimson)]"
              : "bg-[var(--muted)]/20 text-[var(--muted)]"
          }`}
        >
          {player.category}
        </span>
      </div>

      <div className="rounded-xl border-2 border-[var(--gold)]/50 bg-[var(--surface)] p-3 shadow-md">
        {canPush && (
          <div className="mb-2 flex gap-1">
            {[0, 1, 2, 3, 4].map((col) => {
              const isAllowed = canPushColumn(col);
              return (
                <button
                  key={col}
                  onClick={() => onPush(col)}
                  disabled={!isAllowed}
                  className={`flex h-8 w-14 items-center justify-center rounded-md text-xs font-semibold transition-all ${
                    isAllowed
                      ? "bg-[var(--jade)] text-white hover:bg-[var(--jade)]/80"
                      : "bg-[var(--tile-empty)] text-[var(--muted)] cursor-not-allowed"
                  }`}
                >
                  Push
                </button>
              );
            })}
          </div>
        )}

        <div className="grid grid-cols-5 gap-1">
          {player.board.map((row, rowIdx) =>
            row.map((tile, colIdx) => {
              return (
                <div
                  key={`${rowIdx}-${colIdx}`}
                  className={`flex h-16 w-14 items-center justify-center rounded-lg border-2 transition-all ${
                    tile.id !== 0
                      ? "border-[var(--gold)] bg-[var(--tile)] shadow-sm"
                      : "border-dashed border-[var(--gold)]/30 bg-[var(--tile-empty)]"
                  }`}
                >
                  {tile.id !== 0 ? (
                    <TileImage
                      symbol={tile.symbol}
                      alt={tile.symbol}
                      width={48}
                      height={60}
                      className="rounded"
                      fallback={<span className="text-xs text-[var(--muted)]">{tile.symbol}</span>}
                    />
                  ) : null}
                </div>
              );
            })
          )}
        </div>
      </div>
    </div>
  );
}
