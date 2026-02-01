package model

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
