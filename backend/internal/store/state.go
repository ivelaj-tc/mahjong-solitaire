package store

import "mahjong-backend/internal/game"

type GameSnapshot struct {
	Players             []game.Player         `json:"players"`
	CurrentTurn         int                   `json:"currentTurn"`
	Phase               game.GamePhase        `json:"phase"`
	Winner              int                   `json:"winner"`
	SharedTile          *game.Tile            `json:"sharedTile"`
	RPSChoices          map[int]string        `json:"rpsChoices"`
	CategoryChoices     map[int]game.Category `json:"categoryChoices"`
	AvailableCategories []game.Category       `json:"availableCategories"`
	StatusMessage       string                `json:"statusMessage"`
	RemainingTiles      int                   `json:"remainingTiles"`
	TileDeck            []game.Tile           `json:"tileDeck"`
	CategoryFileTypes   map[game.Category]string `json:"categoryFileTypes"`
}

type RoomState struct {
	RoomID   string       `json:"roomId"`
	SourceID string       `json:"sourceId,omitempty"`
	Game     GameSnapshot `json:"game"`
}

func SnapshotFromGame(g *game.Game) GameSnapshot {
	if g == nil {
		return GameSnapshot{}
	}
	return GameSnapshot{
		Players:             clonePlayers(g.Players),
		CurrentTurn:         g.CurrentTurn,
		Phase:               g.Phase,
		Winner:              g.Winner,
		SharedTile:          cloneTilePointer(g.SharedTile),
		RPSChoices:          cloneStringMap(g.RPSChoices),
		CategoryChoices:     cloneCategoryMap(g.CategoryChoices),
		AvailableCategories: append([]game.Category{}, g.AvailableCategories...),
		StatusMessage:       g.StatusMessage,
		RemainingTiles:      g.RemainingTiles,
		TileDeck:            append([]game.Tile{}, g.TileDeck...),
		CategoryFileTypes:   cloneCategoryFileTypes(g.CategoryFileTypes),
	}
}

func GameFromSnapshot(snapshot GameSnapshot, categorySymbols map[game.Category][]string) *game.Game {
	gameState := &game.Game{
		Players:             clonePlayers(snapshot.Players),
		CurrentTurn:         snapshot.CurrentTurn,
		Phase:               snapshot.Phase,
		Winner:              snapshot.Winner,
		SharedTile:          cloneTilePointer(snapshot.SharedTile),
		RPSChoices:          cloneStringMap(snapshot.RPSChoices),
		CategoryChoices:     cloneCategoryMap(snapshot.CategoryChoices),
		AvailableCategories: append([]game.Category{}, snapshot.AvailableCategories...),
		StatusMessage:       snapshot.StatusMessage,
		RemainingTiles:      snapshot.RemainingTiles,
		TileDeck:            append([]game.Tile{}, snapshot.TileDeck...),
		CategorySymbols:     cloneCategorySymbols(categorySymbols),
		CategoryFileTypes:   cloneCategoryFileTypes(snapshot.CategoryFileTypes),
	}
	if gameState.RPSChoices == nil {
		gameState.RPSChoices = make(map[int]string)
	}
	if gameState.CategoryChoices == nil {
		gameState.CategoryChoices = make(map[int]game.Category)
	}
	if gameState.TileDeck == nil {
		gameState.TileDeck = make([]game.Tile, 0)
	}
	if gameState.AvailableCategories == nil {
		gameState.AvailableCategories = make([]game.Category, 0)
	}
	if gameState.CategoryFileTypes == nil {
		gameState.CategoryFileTypes = make(map[game.Category]string)
	}
	if gameState.RemainingTiles == 0 && len(gameState.TileDeck) > 0 {
		gameState.RemainingTiles = len(gameState.TileDeck)
	}
	return gameState
}

func cloneTilePointer(tile *game.Tile) *game.Tile {
	if tile == nil {
		return nil
	}
	copy := *tile
	return &copy
}

func clonePlayers(players []game.Player) []game.Player {
	clone := make([]game.Player, len(players))
	for i, player := range players {
		clone[i] = player
		clone[i].Board = cloneBoard(player.Board)
	}
	return clone
}

func cloneBoard(board [][]game.Tile) [][]game.Tile {
	if board == nil {
		return nil
	}
	clone := make([][]game.Tile, len(board))
	for row := range board {
		clone[row] = append([]game.Tile{}, board[row]...)
	}
	return clone
}

func cloneStringMap(src map[int]string) map[int]string {
	if src == nil {
		return nil
	}
	clone := make(map[int]string, len(src))
	for key, value := range src {
		clone[key] = value
	}
	return clone
}

func cloneCategoryMap(src map[int]game.Category) map[int]game.Category {
	if src == nil {
		return nil
	}
	clone := make(map[int]game.Category, len(src))
	for key, value := range src {
		clone[key] = value
	}
	return clone
}

func cloneCategorySymbols(src map[game.Category][]string) map[game.Category][]string {
	clone := make(map[game.Category][]string, len(src))
	for category, symbols := range src {
		clone[category] = append([]string{}, symbols...)
	}
	return clone
}

func cloneCategoryFileTypes(src map[game.Category]string) map[game.Category]string {
	if src == nil {
		return nil
	}
	clone := make(map[game.Category]string, len(src))
	for category, fileType := range src {
		clone[category] = fileType
	}
	return clone
}
