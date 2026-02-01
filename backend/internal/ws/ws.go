package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"mahjong-backend/internal/game"
	"mahjong-backend/internal/room"
)

type Server struct {
	categorySymbols     map[game.Category][]string
	categoryFileTypes   map[game.Category]string
	availableCategories []game.Category
	waitingRoom         *room.GameRoom
	gamesMux            sync.Mutex
	upgrader            websocket.Upgrader
}

type Client struct {
	Conn     *websocket.Conn
	Room     *room.GameRoom
	PlayerID int
	Send     chan []byte
	server   *Server
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
	Category game.Category `json:"category"`
}

type PushMessage struct {
	Column int `json:"column"`
}

type JoinAck struct {
	PlayerID int    `json:"playerId"`
	RoomID   string `json:"roomId"`
}

func NewServer(
	categorySymbols map[game.Category][]string,
	categoryFileTypes map[game.Category]string,
	availableCategories []game.Category,
) *Server {
	return &Server{
		categorySymbols:     categorySymbols,
		categoryFileTypes:   categoryFileTypes,
		availableCategories: append([]game.Category{}, availableCategories...),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (c *Client) SendChannel() chan []byte {
	return c.Send
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

func (s *Server) Handler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}

	client := &Client{
		Conn:   conn,
		Send:   make(chan []byte, 256),
		server: s,
	}

	go client.writePump()
	go client.readPump()

	log.Println("Client connected")
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
				c.Room.Game.Phase = game.PhaseGameOver
				c.Room.Game.Winner = winnerID
				leaverName := c.Room.Game.Players[c.PlayerID].Name
				winnerName := c.Room.Game.Players[winnerID].Name
				c.Room.Game.StatusMessage = leaverName + " left the game. " + winnerName + " wins!"
			} else {
				c.Room.Game.StatusMessage = "A player disconnected."
				c.Room.Game.Phase = game.PhaseWaiting
				c.Room.Game.Winner = -1
			}
			c.Room.RefreshTimersLocked()
			c.Room.StopBotActionTimerLocked()
			c.Room.Mutex.Unlock()
			c.server.broadcastState(c.Room)
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
			c.server.handleJoin(c, joinMsg)
		case "rps":
			var rpsMsg RPSMessage
			json.Unmarshal(wsMsg.Payload, &rpsMsg)
			c.server.handleRPS(c, rpsMsg)
		case "category":
			var categoryMsg CategoryMessage
			json.Unmarshal(wsMsg.Payload, &categoryMsg)
			c.server.handleCategory(c, categoryMsg)
		case "push":
			var pushMsg PushMessage
			json.Unmarshal(wsMsg.Payload, &pushMsg)
			c.server.handlePush(c, pushMsg)
		case "reset":
			c.server.handleReset(c)
		}
	}
}

func (s *Server) newGameRoom() *room.GameRoom {
	gameState := game.NewGame(s.categorySymbols, s.categoryFileTypes, s.availableCategories)
	gameRoom := room.NewGameRoom(gameState)
	gameRoom.OnStateChange = s.broadcastState
	return gameRoom
}

func (s *Server) handleJoin(c *Client, msg JoinMessage) {
	var gameRoom *room.GameRoom
	if msg.WithBot {
		gameRoom = s.newGameRoom()
	} else {
		s.gamesMux.Lock()
		if s.waitingRoom == nil || len(s.waitingRoom.Game.Players) >= 2 {
			s.waitingRoom = s.newGameRoom()
		}
		gameRoom = s.waitingRoom
		s.gamesMux.Unlock()
	}

	gameRoom.Mutex.Lock()
	playerID := len(gameRoom.Game.Players)
	if playerID >= 2 {
		gameRoom.Mutex.Unlock()
		return
	}

	player := game.Player{
		ID:       playerID,
		Name:     msg.Name,
		Category: game.CategoryBlank,
		Board:    game.EmptyBoard(),
	}
	gameRoom.Game.Players = append(gameRoom.Game.Players, player)
	gameRoom.Clients[c] = true
	c.Room = gameRoom
	c.PlayerID = playerID

	if msg.WithBot && len(gameRoom.Game.Players) < 2 {
		botID := len(gameRoom.Game.Players)
		bot := game.Player{
			ID:       botID,
			Name:     "Bot",
			Category: game.CategoryBlank,
			Board:    game.EmptyBoard(),
		}
		gameRoom.Game.Players = append(gameRoom.Game.Players, bot)
		gameRoom.BotPlayerID = botID
	}

	c.sendMessage("joined", JoinAck{PlayerID: playerID, RoomID: gameRoom.ID})

	if len(gameRoom.Game.Players) == 2 {
		gameRoom.Game.Phase = game.PhaseCategory
		gameRoom.Game.CategoryChoices = make(map[int]game.Category)
		gameRoom.Game.StatusMessage = "Both players joined. Choose your categories."
		gameRoom.RefreshTimersLocked()
		if !msg.WithBot {
			s.gamesMux.Lock()
			if s.waitingRoom == gameRoom {
				s.waitingRoom = nil
			}
			s.gamesMux.Unlock()
		}
	}
	gameRoom.Mutex.Unlock()

	s.broadcastState(gameRoom)
	gameRoom.ScheduleBotAction()
}

func (s *Server) handleRPS(c *Client, msg RPSMessage) {
	if c.Room == nil {
		return
	}
	gameRoom := c.Room
	gameRoom.Mutex.Lock()
	prevPhase := gameRoom.Game.Phase
	gameRoom.Game.HandleRPS(c.PlayerID, msg.Choice)
	gameRoom.SetBotStartDelayLocked(prevPhase)
	gameRoom.RefreshTimersLocked()
	gameRoom.Mutex.Unlock()
	s.broadcastState(gameRoom)
	gameRoom.ScheduleBotAction()
}

func (s *Server) handleCategory(c *Client, msg CategoryMessage) {
	if c.Room == nil {
		return
	}
	gameRoom := c.Room
	gameRoom.Mutex.Lock()
	gameRoom.Game.HandleCategorySelection(c.PlayerID, msg.Category)
	gameRoom.RefreshTimersLocked()
	gameRoom.Mutex.Unlock()
	s.broadcastState(gameRoom)
	gameRoom.ScheduleBotAction()
}

func (s *Server) handlePush(c *Client, msg PushMessage) {
	log.Printf("handlePush: playerID=%d, column=%d", c.PlayerID, msg.Column)
	if c.Room == nil {
		log.Printf("handlePush: no room for client")
		return
	}
	gameRoom := c.Room
	gameRoom.Mutex.Lock()
	gameRoom.Game.PushTile(c.PlayerID, msg.Column)
	gameRoom.RefreshTimersLocked()
	gameRoom.Mutex.Unlock()
	s.broadcastState(gameRoom)
	gameRoom.ScheduleBotAction()
}

func (s *Server) handleReset(c *Client) {
	if c.Room == nil {
		return
	}
	gameRoom := c.Room
	gameRoom.Mutex.Lock()
	players := gameRoom.Game.Players
	gameRoom.Game = game.NewGame(s.categorySymbols, s.categoryFileTypes, s.availableCategories)
	gameRoom.Game.Players = players
	for i := range gameRoom.Game.Players {
		gameRoom.Game.Players[i].Board = game.EmptyBoard()
		gameRoom.Game.Players[i].Category = game.CategoryBlank
	}
	gameRoom.Game.Phase = game.PhaseCategory
	gameRoom.Game.CategoryChoices = make(map[int]game.Category)
	gameRoom.Game.RPSChoices = make(map[int]string)
	gameRoom.Game.StatusMessage = "Rematch! Choose categories, then play RPS."
	gameRoom.BotStartDelayUntil = time.Time{}
	gameRoom.RefreshTimersLocked()
	gameRoom.Mutex.Unlock()
	s.broadcastState(gameRoom)
	gameRoom.ScheduleBotAction()
}

func (s *Server) broadcastState(gameRoom *room.GameRoom) {
	data, _ := json.Marshal(gameRoom.Game)
	msg := WSMessage{Type: "gameState", Payload: data}
	msgBytes, _ := json.Marshal(msg)

	for client := range gameRoom.Clients {
		select {
		case client.SendChannel() <- msgBytes:
		default:
		}
	}
}
