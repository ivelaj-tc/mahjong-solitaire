//go:build legacy
// +build legacy

package main

import (
	crand "crypto/rand"
	"database/sql"
	"encoding/json"
	"log"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	_ "modernc.org/sqlite"
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

type Client struct {
	Conn     *websocket.Conn
	Room     *GameRoom
	PlayerID int
	Send     chan []byte
}

type WSMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type JoinMessage struct {
	Name    string `json:"name"`
	WithBot bool   `json:"withBot"`
}

type RPSMessage struct {
	Choice string `json:"choice"`
}

type CategoryMessage struct {
	Category Category `json:"category"`
}

type PushMessage struct {
	Column int `json:"column"`
}

type JoinAck struct {
	PlayerID int    `json:"playerId"`
	RoomID   string `json:"roomId"`
}

var (
	waitingRoom *GameRoom
	gamesMux    sync.Mutex
	upgrader    = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

const defaultDBPath = "./data/mahjong.db"

var defaultCategorySymbols = map[Category][]string{
	CategoryAnimals: {"panda", "fox", "tiger", "frog", "lion"},
	CategoryFoods:   {"sushi", "dango", "dumpling", "ramen", "tea"},
	CategoryFlowers: {"flower-blue", "flower-green", "flower-orange", "flower-garden", "flower-svgrepo"},
}

var defaultAvailableCategories = []Category{CategoryAnimals, CategoryFoods, CategoryFlowers}

var categorySymbols = cloneCategorySymbols(defaultCategorySymbols)
var availableCategories = append([]Category{}, defaultAvailableCategories...)

func NewGame() *Game {
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
	}
}

func (g *Game) initializeGame(categories []Category) {
	g.TileDeck = make([]Tile, 0)
	g.createTileDeck(categories)
	g.SharedTile = g.drawTile()
	g.Winner = -1
	g.StatusMessage = "Game started!"

	for i := range g.Players {
		g.Players[i].Board = emptyBoard()
	}
	g.resolveTurnForSharedTile()
}

func emptyBoard() [][]Tile {
	board := make([][]Tile, BoardHeight)
	for row := range board {
		board[row] = make([]Tile, BoardWidth)
	}
	return board
}

func initCategoryConfig() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = defaultDBPath
	}

	symbols, categories, err := loadCategoryConfig(dbPath)
	if err != nil {
		log.Printf("Failed to load categories from %s: %v. Using defaults.", dbPath, err)
		categorySymbols = cloneCategorySymbols(defaultCategorySymbols)
		availableCategories = append([]Category{}, defaultAvailableCategories...)
		return
	}
	if len(categories) == 0 {
		log.Printf("No categories found in %s. Using defaults.", dbPath)
		categorySymbols = cloneCategorySymbols(defaultCategorySymbols)
		availableCategories = append([]Category{}, defaultAvailableCategories...)
		return
	}
	categorySymbols = symbols
	availableCategories = categories
	log.Printf("Loaded %d categories from %s", len(categories), dbPath)
}

func loadCategoryConfig(dbPath string) (map[Category][]string, []Category, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, nil, err
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, nil, err
	}
	defer db.Close()

	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, nil, err
	}
	if err := ensureCategoryTables(db); err != nil {
		return nil, nil, err
	}
	seedNeeded, err := isCategorySeedNeeded(db)
	if err != nil {
		return nil, nil, err
	}
	if seedNeeded {
		if err := seedDefaultCategories(db); err != nil {
			return nil, nil, err
		}
	}

	categories, err := fetchCategories(db)
	if err != nil {
		return nil, nil, err
	}
	symbols, err := fetchCategorySymbols(db)
	if err != nil {
		return nil, nil, err
	}
	for _, category := range categories {
		if _, ok := symbols[category]; !ok {
			symbols[category] = []string{}
		}
	}
	return symbols, categories, nil
}

func ensureCategoryTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS categories (
			name TEXT PRIMARY KEY
		);
		CREATE TABLE IF NOT EXISTS category_symbols (
			category TEXT NOT NULL,
			symbol TEXT NOT NULL,
			PRIMARY KEY (category, symbol),
			FOREIGN KEY (category) REFERENCES categories(name) ON DELETE CASCADE
		);
	`)
	return err
}

func isCategorySeedNeeded(db *sql.DB) (bool, error) {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count); err != nil {
		return false, err
	}
	return count == 0, nil
}

func seedDefaultCategories(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, category := range defaultAvailableCategories {
		if _, err := tx.Exec("INSERT OR IGNORE INTO categories (name) VALUES (?)", string(category)); err != nil {
			return err
		}
		for _, symbol := range defaultCategorySymbols[category] {
			if _, err := tx.Exec("INSERT OR IGNORE INTO category_symbols (category, symbol) VALUES (?, ?)", string(category), symbol); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func fetchCategories(db *sql.DB) ([]Category, error) {
	rows, err := db.Query("SELECT name FROM categories ORDER BY rowid")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := make([]Category, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		category := Category(name)
		if category == CategoryBlank {
			continue
		}
		categories = append(categories, category)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return categories, nil
}

func fetchCategorySymbols(db *sql.DB) (map[Category][]string, error) {
	rows, err := db.Query("SELECT category, symbol FROM category_symbols ORDER BY category, symbol")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	symbols := make(map[Category][]string)
	for rows.Next() {
		var categoryName string
		var symbol string
		if err := rows.Scan(&categoryName, &symbol); err != nil {
			return nil, err
		}
		category := Category(categoryName)
		if category == CategoryBlank {
			continue
		}
		symbols[category] = append(symbols[category], symbol)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return symbols, nil
}

func cloneCategorySymbols(src map[Category][]string) map[Category][]string {
	clone := make(map[Category][]string, len(src))
	for category, symbols := range src {
		clone[category] = append([]string{}, symbols...)
	}
	return clone
}

func (g *Game) createTileDeck(categories []Category) {
	tileID := 1
	for _, category := range categories {
		symbols, ok := categorySymbols[category]
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

		// swap
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

func (g *Game) handleRPS(playerID int, choice string) {
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

	g.resolveRPS()
}

func (g *Game) handleCategorySelection(playerID int, category Category) {
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

func (g *Game) resolveRPS() {
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

func (g *Game) autoSelectCategories() bool {
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

func pickBotCategory(game *Game) (Category, bool) {
	if game == nil {
		return "", false
	}
	chosen := make(map[Category]bool)
	for _, category := range game.CategoryChoices {
		chosen[category] = true
	}
	return pickRandomCategory(game.AvailableCategories, chosen)
}

func (g *Game) autoSelectRPS() bool {
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

func (g *Game) autoPlayTurn() bool {
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
	g.pushTile(playerID, column)
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
		for row := 0; row < BoardHeight; row++ {
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
		for row := 0; row < BoardHeight; row++ {
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
	for row := 0; row < BoardHeight; row++ {
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

func (g *Game) pushTile(playerID, column int) {
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

func (room *GameRoom) refreshTimersLocked() {
	switch room.Game.Phase {
	case PhaseCategory:
		room.resetCategoryTimerLocked()
		room.stopRPSTimerLocked()
		room.stopTurnTimerLocked()
	case PhaseRPS:
		room.stopCategoryTimerLocked()
		room.resetRPSTimerLocked()
		room.stopTurnTimerLocked()
	case PhasePlaying:
		room.stopCategoryTimerLocked()
		room.stopRPSTimerLocked()
		room.resetTurnTimerLocked()
	default:
		room.stopCategoryTimerLocked()
		room.stopRPSTimerLocked()
		room.stopTurnTimerLocked()
	}
}

func (room *GameRoom) stopCategoryTimerLocked() {
	room.CategoryTimerID++
	if room.CategoryTimer != nil {
		room.CategoryTimer.Stop()
		room.CategoryTimer = nil
	}
}

func (room *GameRoom) stopRPSTimerLocked() {
	room.RPSTimerID++
	if room.RPSTimer != nil {
		room.RPSTimer.Stop()
		room.RPSTimer = nil
	}
}

func (room *GameRoom) stopTurnTimerLocked() {
	room.TurnTimerID++
	if room.TurnTimer != nil {
		room.TurnTimer.Stop()
		room.TurnTimer = nil
	}
}

func (room *GameRoom) setBotStartDelayLocked(prevPhase GamePhase) {
	if room.Game == nil {
		return
	}
	if prevPhase != PhasePlaying && room.Game.Phase == PhasePlaying {
		if room.BotPlayerID >= 0 && room.Game.CurrentTurn == room.BotPlayerID {
			room.BotStartDelayUntil = time.Now().Add(BotFirstTurnDelay)
		} else {
			room.BotStartDelayUntil = time.Time{}
		}
	}
}

func (room *GameRoom) stopBotActionTimerLocked() {
	room.BotActionID++
	if room.BotActionTimer != nil {
		room.BotActionTimer.Stop()
		room.BotActionTimer = nil
	}
}

func (room *GameRoom) shouldBotActLocked() bool {
	if room.Game == nil {
		return false
	}
	if room.BotPlayerID < 0 || room.BotPlayerID >= len(room.Game.Players) {
		return false
	}
	switch room.Game.Phase {
	case PhaseCategory:
		_, chosen := room.Game.CategoryChoices[room.BotPlayerID]
		return !chosen
	case PhaseRPS:
		_, chosen := room.Game.RPSChoices[room.BotPlayerID]
		return !chosen
	case PhasePlaying:
		return room.Game.CurrentTurn == room.BotPlayerID && room.Game.SharedTile != nil
	default:
		return false
	}
}

func (room *GameRoom) performBotActionLocked() {
	if room.Game == nil {
		return
	}
	if room.BotPlayerID < 0 || room.BotPlayerID >= len(room.Game.Players) {
		return
	}
	switch room.Game.Phase {
	case PhaseCategory:
		if _, chosen := room.Game.CategoryChoices[room.BotPlayerID]; chosen {
			return
		}
		category, ok := pickBotCategory(room.Game)
		if !ok {
			return
		}
		room.Game.handleCategorySelection(room.BotPlayerID, category)
	case PhaseRPS:
		if _, chosen := room.Game.RPSChoices[room.BotPlayerID]; chosen {
			return
		}
		choices := []string{"rock", "paper", "scissors"}
		choice := choices[rand.Intn(len(choices))]
		room.Game.handleRPS(room.BotPlayerID, choice)
	case PhasePlaying:
		if room.Game.CurrentTurn != room.BotPlayerID {
			return
		}
		room.BotStartDelayUntil = time.Time{}
		room.Game.autoPlayTurn()
		if room.Game.Phase == PhasePlaying && room.Game.CurrentTurn == room.BotPlayerID && room.Game.SharedTile != nil {
			room.BotStartDelayUntil = time.Now().Add(BotConsecutiveTurnDelay)
		}
	}
}

func (room *GameRoom) scheduleBotAction() {
	if room == nil {
		return
	}
	room.Mutex.Lock()
	if room.BotPlayerID < 0 || room.Game == nil {
		room.stopBotActionTimerLocked()
		room.Mutex.Unlock()
		return
	}
	if !room.shouldBotActLocked() {
		room.stopBotActionTimerLocked()
		room.Mutex.Unlock()
		return
	}
	room.BotActionID++
	id := room.BotActionID
	if room.BotActionTimer != nil {
		room.BotActionTimer.Stop()
	}
	delay := BotActionDelay
	if room.Game.Phase == PhasePlaying && room.Game.CurrentTurn == room.BotPlayerID && !room.BotStartDelayUntil.IsZero() {
		remaining := time.Until(room.BotStartDelayUntil)
		if remaining > delay {
			delay = remaining
		} else if remaining <= 0 {
			room.BotStartDelayUntil = time.Time{}
		}
	}
	room.BotActionTimer = time.AfterFunc(delay, func() {
		room.Mutex.Lock()
		if id != room.BotActionID {
			room.Mutex.Unlock()
			return
		}
		room.performBotActionLocked()
		room.refreshTimersLocked()
		room.Mutex.Unlock()
		broadcastState(room)
		room.scheduleBotAction()
	})
	room.Mutex.Unlock()
}

func (room *GameRoom) resetCategoryTimerLocked() {
	room.CategoryTimerID++
	id := room.CategoryTimerID
	if room.CategoryTimer != nil {
		room.CategoryTimer.Stop()
	}
	room.CategoryTimer = time.AfterFunc(CategoryTimeout, func() {
		room.Mutex.Lock()
		if id != room.CategoryTimerID || room.Game.Phase != PhaseCategory {
			room.Mutex.Unlock()
			return
		}
		changed := room.Game.autoSelectCategories()
		if changed && len(room.Game.CategoryChoices) == 2 {
			room.Game.Phase = PhaseRPS
			room.Game.RPSChoices = make(map[int]string)
			room.Game.StatusMessage = "Categories auto-selected. Play RPS!"
		}
		room.refreshTimersLocked()
		room.Mutex.Unlock()
		broadcastState(room)
		room.scheduleBotAction()
	})
}

func (room *GameRoom) resetRPSTimerLocked() {
	room.RPSTimerID++
	id := room.RPSTimerID
	if room.RPSTimer != nil {
		room.RPSTimer.Stop()
	}
	room.RPSTimer = time.AfterFunc(RPSTimeout, func() {
		room.Mutex.Lock()
		if id != room.RPSTimerID || room.Game.Phase != PhaseRPS {
			room.Mutex.Unlock()
			return
		}
		prevPhase := room.Game.Phase
		if room.Game.autoSelectRPS() {
			room.Game.resolveRPS()
		}
		room.setBotStartDelayLocked(prevPhase)
		room.refreshTimersLocked()
		room.Mutex.Unlock()
		broadcastState(room)
		room.scheduleBotAction()
	})
}

func (room *GameRoom) resetTurnTimerLocked() {
	room.TurnTimerID++
	id := room.TurnTimerID
	if room.TurnTimer != nil {
		room.TurnTimer.Stop()
	}
	room.TurnTimer = time.AfterFunc(TurnTimeout, func() {
		room.Mutex.Lock()
		if id != room.TurnTimerID || room.Game.Phase != PhasePlaying {
			room.Mutex.Unlock()
			return
		}
		room.Game.autoPlayTurn()
		room.refreshTimersLocked()
		room.Mutex.Unlock()
		broadcastState(room)
		room.scheduleBotAction()
	})
}

func newGameRoom() *GameRoom {
	roomID := uuid.Must(uuid.NewV4()).String()
	return &GameRoom{
		ID:          roomID,
		Game:        NewGame(),
		Clients:     make(map[*Client]bool),
		BotPlayerID: -1,
	}
}

func (c *Client) sendMessage(msgType string, payload interface{}) {
	data, _ := json.Marshal(payload)
	msg := WSMessage{Type: msgType, Payload: data}
	msgBytes, _ := json.Marshal(msg)
	select {
	case c.Send <- msgBytes:
	default:
	}
}

func broadcastState(room *GameRoom) {
	data, _ := json.Marshal(room.Game)
	msg := WSMessage{Type: "gameState", Payload: data}
	msgBytes, _ := json.Marshal(msg)

	for client := range room.Clients {
		select {
		case client.Send <- msgBytes:
		default:
		}
	}
}

func (c *Client) writePump() {
	defer c.Conn.Close()
	for msg := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		if c.Room != nil {
			c.Room.Mutex.Lock()
			delete(c.Room.Clients, c)
			if c.PlayerID >= 0 && c.PlayerID < len(c.Room.Game.Players) && len(c.Room.Game.Players) > 1 {
				winnerID := (c.PlayerID + 1) % 2
				c.Room.Game.Phase = PhaseGameOver
				c.Room.Game.Winner = winnerID
				leaverName := c.Room.Game.Players[c.PlayerID].Name
				winnerName := c.Room.Game.Players[winnerID].Name
				c.Room.Game.StatusMessage = leaverName + " left the game. " + winnerName + " wins!"
			} else {
				c.Room.Game.StatusMessage = "A player disconnected."
				c.Room.Game.Phase = PhaseWaiting
				c.Room.Game.Winner = -1
			}
			c.Room.refreshTimersLocked()
			c.Room.stopBotActionTimerLocked()
			c.Room.Mutex.Unlock()
			broadcastState(c.Room)
		}
		close(c.Send)
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			return
		}

		var wsMsg WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Println("JSON error:", err)
			continue
		}

		switch wsMsg.Type {
		case "join":
			var joinMsg JoinMessage
			json.Unmarshal(wsMsg.Payload, &joinMsg)
			handleJoin(c, joinMsg)

		case "rps":
			var rpsMsg RPSMessage
			json.Unmarshal(wsMsg.Payload, &rpsMsg)
			handleRPS(c, rpsMsg)

		case "category":
			var categoryMsg CategoryMessage
			json.Unmarshal(wsMsg.Payload, &categoryMsg)
			handleCategory(c, categoryMsg)

		case "push":
			var pushMsg PushMessage
			json.Unmarshal(wsMsg.Payload, &pushMsg)
			handlePush(c, pushMsg)

		case "reset":
			handleReset(c)
		}
	}
}

func handleJoin(c *Client, msg JoinMessage) {
	var room *GameRoom
	if msg.WithBot {
		room = newGameRoom()
	} else {
		gamesMux.Lock()
		if waitingRoom == nil || len(waitingRoom.Game.Players) >= 2 {
			waitingRoom = newGameRoom()
		}
		room = waitingRoom
		gamesMux.Unlock()
	}

	room.Mutex.Lock()
	playerID := len(room.Game.Players)
	if playerID >= 2 {
		room.Mutex.Unlock()
		return
	}

	player := Player{
		ID:       playerID,
		Name:     msg.Name,
		Category: CategoryBlank,
		Board:    emptyBoard(),
	}
	room.Game.Players = append(room.Game.Players, player)
	room.Clients[c] = true
	c.Room = room
	c.PlayerID = playerID

	if msg.WithBot && len(room.Game.Players) < 2 {
		botID := len(room.Game.Players)
		bot := Player{
			ID:       botID,
			Name:     "Bot",
			Category: CategoryBlank,
			Board:    emptyBoard(),
		}
		room.Game.Players = append(room.Game.Players, bot)
		room.BotPlayerID = botID
	}

	c.sendMessage("joined", JoinAck{PlayerID: playerID, RoomID: room.ID})

	if len(room.Game.Players) == 2 {
		room.Game.Phase = PhaseCategory
		room.Game.CategoryChoices = make(map[int]Category)
		room.Game.StatusMessage = "Both players joined. Choose your categories."
		room.refreshTimersLocked()
		if !msg.WithBot {
			gamesMux.Lock()
			if waitingRoom == room {
				waitingRoom = nil
			}
			gamesMux.Unlock()
		}
	}
	room.Mutex.Unlock()

	broadcastState(room)
	room.scheduleBotAction()
}

func handleRPS(c *Client, msg RPSMessage) {
	if c.Room == nil {
		return
	}
	room := c.Room
	room.Mutex.Lock()
	prevPhase := room.Game.Phase
	room.Game.handleRPS(c.PlayerID, msg.Choice)
	room.setBotStartDelayLocked(prevPhase)
	room.refreshTimersLocked()
	room.Mutex.Unlock()
	broadcastState(room)
	room.scheduleBotAction()
}

func handleCategory(c *Client, msg CategoryMessage) {
	if c.Room == nil {
		return
	}
	room := c.Room
	room.Mutex.Lock()
	room.Game.handleCategorySelection(c.PlayerID, msg.Category)
	room.refreshTimersLocked()
	room.Mutex.Unlock()
	broadcastState(room)
	room.scheduleBotAction()
}

func handlePush(c *Client, msg PushMessage) {
	log.Printf("handlePush: playerID=%d, column=%d", c.PlayerID, msg.Column)
	if c.Room == nil {
		log.Printf("handlePush: no room for client")
		return
	}
	room := c.Room
	room.Mutex.Lock()
	room.Game.pushTile(c.PlayerID, msg.Column)
	room.refreshTimersLocked()
	room.Mutex.Unlock()
	broadcastState(room)
	room.scheduleBotAction()
}

func handleReset(c *Client) {
	if c.Room == nil {
		return
	}
	room := c.Room
	room.Mutex.Lock()
	players := room.Game.Players
	room.Game = NewGame()
	room.Game.Players = players
	for i := range room.Game.Players {
		room.Game.Players[i].Board = emptyBoard()
		room.Game.Players[i].Category = CategoryBlank
	}
	room.Game.Phase = PhaseCategory
	room.Game.CategoryChoices = make(map[int]Category)
	room.Game.RPSChoices = make(map[int]string)
	room.Game.StatusMessage = "Rematch! Choose categories, then play RPS."
	room.BotStartDelayUntil = time.Time{}
	room.refreshTimersLocked()
	room.Mutex.Unlock()
	broadcastState(room)
	room.scheduleBotAction()
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	client := &Client{
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	go client.writePump()
	go client.readPump()

	log.Println("Client connected")
}

func main() {
	rand.Seed(time.Now().UnixNano())
	initCategoryConfig()

	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	log.Println("Mahjong WebSocket server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
