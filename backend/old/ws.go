//go:build legacy
// +build legacy

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

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
