package game

import (
	crand "crypto/rand"
	"log"
	"math/big"
	"math/rand"

	"mahjong-backend/internal/model"
)

const (
	BoardWidth  = 5
	BoardHeight = 5
)

type Category = model.Category

const (
	CategoryAnimals = model.CategoryAnimals
	CategoryFoods   = model.CategoryFoods
	CategoryFlowers = model.CategoryFlowers
	CategoryBlank   = model.CategoryBlank
)

type Tile = model.Tile
type Player = model.Player

type GamePhase = model.GamePhase

const (
	PhaseWaiting  = model.PhaseWaiting
	PhaseCategory = model.PhaseCategory
	PhaseRPS      = model.PhaseRPS
	PhasePlaying  = model.PhasePlaying
	PhaseGameOver = model.PhaseGameOver
)

type Game struct {
	Players             []Player              `json:"players"`
	CurrentTurn         int                   `json:"currentTurn"`
	Phase               GamePhase             `json:"phase"`
	Winner              int                   `json:"winner"`
	SharedTile          *Tile                 `json:"sharedTile"`
	RPSChoices          map[int]string        `json:"rpsChoices"`
	CategoryChoices     map[int]Category      `json:"categoryChoices"`
	AvailableCategories []Category            `json:"availableCategories"`
	StatusMessage       string                `json:"statusMessage"`
	RemainingTiles      int                   `json:"remainingTiles"`
	TileDeck            []Tile                `json:"-"`
	CategorySymbols     map[Category][]string `json:"-"`
	CategoryFileTypes   map[Category]string   `json:"categoryFileTypes"`
}

func NewGame(
	categorySymbols map[Category][]string,
	categoryFileTypes map[Category]string,
	availableCategories []Category,
) *Game {
	return &Game{
		Players:             make([]Player, 0, 2),
		Phase:               PhaseWaiting,
		Winner:              -1,
		CurrentTurn:         0,
		RPSChoices:          make(map[int]string),
		CategoryChoices:     make(map[int]Category),
		AvailableCategories: append([]Category{}, availableCategories...),
		StatusMessage:       "Waiting for players...",
		RemainingTiles:      0,
		TileDeck:            make([]Tile, 0),
		CategorySymbols:     cloneCategorySymbols(categorySymbols),
		CategoryFileTypes:   cloneCategoryFileTypes(categoryFileTypes),
	}
}

func (g *Game) initializeGame(categories []Category) {
	g.TileDeck = make([]Tile, 0)
	g.createTileDeck(categories)
	g.SharedTile = g.drawTile()
	g.Winner = -1
	g.StatusMessage = "Game started!"

	for i := range g.Players {
		g.Players[i].Board = EmptyBoard()
	}
	g.resolveTurnForSharedTile()
}

func EmptyBoard() [][]Tile {
	board := make([][]Tile, BoardHeight)
	for row := range board {
		board[row] = make([]Tile, BoardWidth)
	}
	return board
}

func cloneCategorySymbols(src map[Category][]string) map[Category][]string {
	clone := make(map[Category][]string, len(src))
	for category, symbols := range src {
		clone[category] = append([]string{}, symbols...)
	}
	return clone
}

func cloneCategoryFileTypes(src map[Category]string) map[Category]string {
	clone := make(map[Category]string, len(src))
	for category, fileType := range src {
		clone[category] = fileType
	}
	return clone
}

func (g *Game) createTileDeck(categories []Category) {
	tileID := 1
	for _, category := range categories {
		symbols, ok := g.CategorySymbols[category]
		if !ok {
			log.Printf("Missing symbols for category: %s", category)
			continue
		}
		for _, symbol := range symbols {
			for j := 0; j < BoardHeight; j++ {
				g.TileDeck = append(g.TileDeck, Tile{ID: tileID, Category: category, Symbol: symbol})
				tileID++
			}
		}
	}
	g.TileDeck = append(g.TileDeck, Tile{ID: tileID, Category: CategoryBlank, Symbol: "blank"})
	shuffleTiles(g.TileDeck)
	g.RemainingTiles = len(g.TileDeck)
}

func (g *Game) isCategoryAvailable(category Category) bool {
	for _, available := range g.AvailableCategories {
		if available == category {
			return true
		}
	}
	return false
}

func shuffleTiles(tiles []Tile) error {
	n := len(tiles)
	for i := n - 1; i > 0; i-- {
		jBig, err := crand.Int(crand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		j := int(jBig.Int64())
		tiles[i], tiles[j] = tiles[j], tiles[i]
	}
	return nil
}

func (g *Game) drawTile() *Tile {
	if len(g.TileDeck) == 0 {
		g.RemainingTiles = 0
		return nil
	}
	tile := g.TileDeck[0]
	g.TileDeck = g.TileDeck[1:]
	g.RemainingTiles = len(g.TileDeck)
	return &tile
}

func (g *Game) HandleRPS(playerID int, choice string) {
	if g.Phase != PhaseRPS {
		return
	}
	if len(g.CategoryChoices) < 2 {
		g.StatusMessage = "Select categories before playing RPS."
		return
	}

	g.RPSChoices[playerID] = choice
	if len(g.RPSChoices) < 2 {
		return
	}

	g.ResolveRPS()
}

func (g *Game) HandleCategorySelection(playerID int, category Category) {
	if g.Phase != PhaseCategory {
		return
	}
	if !g.isCategoryAvailable(category) {
		g.StatusMessage = "That category is not available."
		return
	}
	for _, chosen := range g.CategoryChoices {
		if chosen == category {
			g.StatusMessage = "That category is already chosen."
			return
		}
	}

	g.CategoryChoices[playerID] = category
	if len(g.CategoryChoices) < 2 {
		g.StatusMessage = g.Players[playerID].Name + " selected a category. Waiting for opponent."
		return
	}

	g.Phase = PhaseRPS
	g.RPSChoices = make(map[int]string)
	g.StatusMessage = "Categories selected. Play RPS!"
}

func determineRPSWinner(choice1, choice2 string) int {
	if choice1 == choice2 {
		return -1
	}

	wins := map[string]string{"rock": "scissors", "paper": "rock", "scissors": "paper"}
	if wins[choice1] == choice2 {
		return 0
	}
	return 1
}

func (g *Game) ResolveRPS() {
	p1Choice := g.RPSChoices[0]
	p2Choice := g.RPSChoices[1]
	winner := determineRPSWinner(p1Choice, p2Choice)

	if winner == -1 {
		g.RPSChoices = make(map[int]string)
		g.StatusMessage = "Tie! Choose again."
		return
	}

	choice0, ok0 := g.CategoryChoices[0]
	choice1, ok1 := g.CategoryChoices[1]
	if !ok0 || !ok1 {
		g.StatusMessage = "Waiting for both category selections."
		g.RPSChoices = make(map[int]string)
		return
	}

	g.CurrentTurn = winner
	if winner == 0 {
		g.Players[0].Category = choice0
		g.Players[1].Category = choice1
	} else {
		g.Players[0].Category = choice1
		g.Players[1].Category = choice0
	}

	g.initializeGame([]Category{g.Players[0].Category, g.Players[1].Category})
	g.Phase = PhasePlaying
	g.StatusMessage = g.Players[winner].Name + " won RPS and starts."
	g.resolveTurnForSharedTile()
}

func (g *Game) AutoSelectCategories() bool {
	if g.Phase != PhaseCategory {
		return false
	}
	if len(g.AvailableCategories) < 2 {
		g.StatusMessage = "Not enough categories to auto-assign."
		return false
	}

	chosen := make(map[Category]bool)
	for _, category := range g.CategoryChoices {
		chosen[category] = true
	}

	for playerID := 0; playerID < 2; playerID++ {
		if _, ok := g.CategoryChoices[playerID]; ok {
			continue
		}
		category, ok := pickRandomCategory(g.AvailableCategories, chosen)
		if !ok {
			g.StatusMessage = "Not enough categories to auto-assign."
			return false
		}
		g.CategoryChoices[playerID] = category
		chosen[category] = true
	}

	if len(g.CategoryChoices) == 2 {
		g.StatusMessage = "Categories auto-selected. Play RPS!"
		return true
	}
	return false
}

func pickRandomCategory(categories []Category, used map[Category]bool) (Category, bool) {
	choices := make([]Category, 0, len(categories))
	for _, category := range categories {
		if !used[category] {
			choices = append(choices, category)
		}
	}
	if len(choices) == 0 {
		return "", false
	}
	return choices[rand.Intn(len(choices))], true
}

func PickBotCategory(game *Game) (Category, bool) {
	if game == nil {
		return "", false
	}
	chosen := make(map[Category]bool)
	for _, category := range game.CategoryChoices {
		chosen[category] = true
	}
	return pickRandomCategory(game.AvailableCategories, chosen)
}

func (g *Game) AutoSelectRPS() bool {
	if g.Phase != PhaseRPS {
		return false
	}
	choices := []string{"rock", "paper", "scissors"}
	choice0, ok0 := g.RPSChoices[0]
	choice1, ok1 := g.RPSChoices[1]

	if !ok0 && !ok1 {
		choice0 = choices[rand.Intn(len(choices))]
		g.RPSChoices[0] = choice0
		g.RPSChoices[1] = randomRPSChoiceExcluding(choice0)
		return true
	}

	if !ok0 {
		g.RPSChoices[0] = randomRPSChoiceExcluding(choice1)
	}
	if !ok1 {
		g.RPSChoices[1] = randomRPSChoiceExcluding(choice0)
	}

	return len(g.RPSChoices) == 2
}

func randomRPSChoiceExcluding(exclude string) string {
	choices := []string{"rock", "paper", "scissors"}
	available := make([]string, 0, len(choices)-1)
	for _, choice := range choices {
		if choice != exclude {
			available = append(available, choice)
		}
	}
	return available[rand.Intn(len(available))]
}

func (g *Game) AutoPlayTurn() bool {
	if g.Phase != PhasePlaying || g.SharedTile == nil || len(g.Players) < 2 {
		return false
	}
	playerID := g.CurrentTurn
	if playerID < 0 || playerID >= len(g.Players) {
		return false
	}
	column := g.findMatchingColumn(playerID, g.SharedTile)
	if column == -1 {
		log.Printf("Auto-move skipped: no valid columns for player %d", playerID)
		g.resolveTurnForSharedTile()
		return false
	}
	log.Printf("Auto-move: playerID=%d, column=%d", playerID, column)
	g.PushTile(playerID, column)
	return true
}

func (g *Game) findMatchingColumn(playerID int, tile *Tile) int {
	if tile == nil || playerID < 0 || playerID >= len(g.Players) {
		return -1
	}
	if !tileMatchesCategory(tile, g.Players[playerID].Category) {
		return -1
	}
	if tile.Category == CategoryBlank {
		board := g.Players[playerID].Board
		emptyColumns := make([]int, 0)
		for col := 0; col < BoardWidth; col++ {
			if board[0][col].ID == 0 {
				emptyColumns = append(emptyColumns, col)
			}
		}
		if len(emptyColumns) == 1 {
			return emptyColumns[0]
		}
		if len(emptyColumns) > 1 {
			return emptyColumns[rand.Intn(len(emptyColumns))]
		}
		return rand.Intn(BoardWidth)
	}
	board := g.Players[playerID].Board
	matching := make([]int, 0)
	empty := make([]int, 0)
	for col := 0; col < BoardWidth; col++ {
		anchorFound := false
		var anchor Tile
		for row := BoardHeight - 1; row >= 0; row-- {
			cell := board[row][col]
			if cell.ID != 0 {
				anchor = cell
				anchorFound = true
				break
			}
		}
		if !anchorFound {
			empty = append(empty, col)
			continue
		}
		if anchor.Category == CategoryBlank || anchor.Symbol == tile.Symbol {
			matching = append(matching, col)
		}
	}
	if len(matching) > 0 {
		return matching[rand.Intn(len(matching))]
	}
	if len(empty) > 0 {
		return empty[rand.Intn(len(empty))]
	}
	return -1
}

func (g *Game) canPushColumn(playerID, column int, tile *Tile) bool {
	if tile == nil || playerID < 0 || playerID >= len(g.Players) {
		return false
	}
	if column < 0 || column >= BoardWidth {
		return false
	}
	if !tileMatchesCategory(tile, g.Players[playerID].Category) {
		return false
	}
	if tile.Category == CategoryBlank {
		return true
	}
	board := g.Players[playerID].Board
	hasMatchingFaceup := false
	for col := 0; col < BoardWidth; col++ {
		matchFound := false
		var matchAnchor Tile
		for row := BoardHeight - 1; row >= 0; row-- {
			cell := board[row][col]
			if cell.ID != 0 {
				matchAnchor = cell
				matchFound = true
				break
			}
		}
		if matchFound && (matchAnchor.Category == CategoryBlank || matchAnchor.Symbol == tile.Symbol) {
			hasMatchingFaceup = true
			break
		}
	}
	columnAnchorFound := false
	var columnAnchor Tile
	for row := BoardHeight - 1; row >= 0; row-- {
		cell := board[row][column]
		if cell.ID != 0 {
			columnAnchor = cell
			columnAnchorFound = true
			break
		}
	}
	if !columnAnchorFound {
		return !hasMatchingFaceup
	}
	if columnAnchor.Category == CategoryBlank {
		return true
	}
	return columnAnchor.Symbol == tile.Symbol
}

func (g *Game) hasValidMove(playerID int, tile *Tile) bool {
	if tile == nil || playerID < 0 || playerID >= len(g.Players) {
		return false
	}
	for col := 0; col < BoardWidth; col++ {
		if g.canPushColumn(playerID, col, tile) {
			return true
		}
	}
	return false
}

func (g *Game) PushTile(playerID, column int) {
	log.Printf("pushTile called: playerID=%d, column=%d, phase=%s, currentTurn=%d, sharedTile=%v",
		playerID, column, g.Phase, g.CurrentTurn, g.SharedTile)

	if g.Phase != PhasePlaying {
		log.Printf("Push rejected: phase is %s, not playing", g.Phase)
		return
	}
	if g.CurrentTurn != playerID {
		log.Printf("Push rejected: not player's turn (currentTurn=%d, playerID=%d)", g.CurrentTurn, playerID)
		return
	}
	if g.SharedTile == nil {
		log.Printf("Push rejected: no shared tile")
		return
	}
	if column < 0 || column >= BoardWidth {
		log.Printf("Push rejected: invalid column %d", column)
		return
	}

	player := &g.Players[playerID]
	pushedTile := *g.SharedTile
	matchesCategory := tileMatchesCategory(&pushedTile, player.Category)

	if !matchesCategory {
		opponentID := (playerID + 1) % 2
		g.CurrentTurn = opponentID
		g.StatusMessage = g.Players[playerID].Name + " passed the tile to " + g.Players[opponentID].Name + "."
		return
	}

	if !g.canPushColumn(playerID, column, &pushedTile) {
		g.StatusMessage = "Choose a column with a matching faceup tile or an empty column."
		return
	}

	pushedOut := pushIntoBoard(player.Board, column, pushedTile)
	log.Printf("Tile pushed out from top: id=%d, category=%s, symbol=%s", pushedOut.ID, pushedOut.Category, pushedOut.Symbol)

	if pushedOut.ID != 0 {
		g.SharedTile = &pushedOut
	} else {
		g.SharedTile = g.drawTile()
	}

	g.StatusMessage = player.Name + " pushed a tile."
	g.checkWinCondition()

	if g.Phase == PhasePlaying && g.SharedTile == nil {
		g.Phase = PhaseGameOver
		g.Winner = -1
		g.StatusMessage = "No more tiles! Game is a draw."
		log.Printf("Game ended: no more tiles")
		return
	}

	if g.Phase == PhasePlaying {
		g.resolveTurnForSharedTile()
	}
}

func pushIntoBoard(board [][]Tile, column int, tile Tile) Tile {
	pushedOut := board[0][column]
	for row := 0; row < BoardHeight-1; row++ {
		board[row][column] = board[row+1][column]
	}
	board[BoardHeight-1][column] = tile
	return pushedOut
}

func (g *Game) checkWinCondition() {
	for i := range g.Players {
		if g.hasUniformColumns(g.Players[i].Board) {
			g.Phase = PhaseGameOver
			g.Winner = i
			g.StatusMessage = g.Players[i].Name + " completed all columns!"
			return
		}
	}
}

func (g *Game) hasUniformColumns(board [][]Tile) bool {
	for col := 0; col < BoardWidth; col++ {
		first := board[0][col]
		if first.ID == 0 || first.Category == CategoryBlank {
			return false
		}
		for row := 1; row < BoardHeight; row++ {
			tile := board[row][col]
			if tile.ID == 0 || tile.Category == CategoryBlank || tile.Symbol != first.Symbol {
				return false
			}
		}
	}
	return true
}

func tileMatchesCategory(tile *Tile, category Category) bool {
	if tile == nil {
		return false
	}
	if tile.Category == CategoryBlank {
		return true
	}
	return tile.Category == category
}

func (g *Game) resolveTurnForSharedTile() {
	if g.Phase != PhasePlaying || g.SharedTile == nil || len(g.Players) < 2 {
		return
	}
	for {
		if g.hasValidMove(g.CurrentTurn, g.SharedTile) {
			return
		}
		opponent := (g.CurrentTurn + 1) % 2
		if g.hasValidMove(opponent, g.SharedTile) {
			g.StatusMessage = g.Players[g.CurrentTurn].Name + " passed the tile to " + g.Players[opponent].Name + "."
			g.CurrentTurn = opponent
			return
		}
		g.SharedTile = g.drawTile()
		if g.SharedTile == nil {
			g.Phase = PhaseGameOver
			g.Winner = -1
			g.StatusMessage = "No more tiles! Game is a draw."
			log.Printf("Game ended: no more tiles")
			return
		}
		g.StatusMessage = "No matching columns. Drawing a new tile."
	}
}
