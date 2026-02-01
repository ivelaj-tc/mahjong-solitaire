export type Category = string;
export type GamePhase = "waiting" | "category" | "rps" | "playing" | "gameover";
export type ThemeMode = "light" | "dark";

export interface Tile {
  id: number;
  category: Category;
  symbol: string;
}

export interface Player {
  id: number;
  name: string;
  category: Category;
  board: Tile[][];
}

export interface GameState {
  players: Player[];
  currentTurn: number;
  phase: GamePhase;
  winner: number;
  sharedTile: Tile | null;
  rpsChoices: Record<number, string>;
  categoryChoices: Record<string, Category>;
  availableCategories: Category[];
  statusMessage: string;
  remainingTiles: number;
}
