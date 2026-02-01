//go:build legacy
// +build legacy

package main

import (
	"sync"
	"time"
)

const (
	BoardWidth              = 5
	BoardHeight             = 5
	CategoryTimeout         = 5 * time.Second
	RPSTimeout              = 5 * time.Second
	TurnTimeout             = 15 * time.Second
	BotActionDelay          = 900 * time.Millisecond
	BotFirstTurnDelay       = 4 * time.Second
	BotConsecutiveTurnDelay = 2 * time.Second
)

type Category string

const (
	CategoryAnimals Category = "animals"
	CategoryFoods   Category = "foods"
	CategoryFlowers Category = "flowers"
	CategoryBlank   Category = "blank"
)

type Tile struct {
	ID       int      `json:"id"`
	Category Category `json:"category"`
	Symbol   string   `json:"symbol"`
}

type Player struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Category Category `json:"category"`
	Board    [][]Tile `json:"board"`
}

type GamePhase string

const (
	PhaseWaiting  GamePhase = "waiting"
	PhaseCategory GamePhase = "category"
	PhaseRPS      GamePhase = "rps"
	PhasePlaying  GamePhase = "playing"
	PhaseGameOver GamePhase = "gameover"
)

type Game struct {
	Players             []Player         `json:"players"`
	CurrentTurn         int              `json:"currentTurn"`
	Phase               GamePhase        `json:"phase"`
	Winner              int              `json:"winner"`
	SharedTile          *Tile            `json:"sharedTile"`
	RPSChoices          map[int]string   `json:"rpsChoices"`
	CategoryChoices     map[int]Category `json:"categoryChoices"`
	AvailableCategories []Category       `json:"availableCategories"`
	StatusMessage       string           `json:"statusMessage"`
	RemainingTiles      int              `json:"remainingTiles"`
	TileDeck            []Tile           `json:"-"`
}

type GameRoom struct {
	ID                 string
	Game               *Game
	Clients            map[*Client]bool
	Mutex              sync.Mutex
	CategoryTimer      *time.Timer
	RPSTimer           *time.Timer
	TurnTimer          *time.Timer
	CategoryTimerID    int
	RPSTimerID         int
	TurnTimerID        int
	BotPlayerID        int
	BotActionTimer     *time.Timer
	BotActionID        int
	BotStartDelayUntil time.Time
}
