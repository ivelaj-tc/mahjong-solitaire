package room

import (
	"math/rand"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"mahjong-backend/internal/game"
)

const (
	CategoryTimeout         = 5 * time.Second
	RPSTimeout              = 5 * time.Second
	TurnTimeout             = 15 * time.Second
	BotActionDelay          = 900 * time.Millisecond
	BotFirstTurnDelay       = 4 * time.Second
	BotConsecutiveTurnDelay = 2 * time.Second
)

type Client interface {
	SendChannel() chan []byte
}

type GameRoom struct {
	ID                 string
	Game               *game.Game
	Clients            map[Client]bool
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
	OnStateChange      func(*GameRoom)
}

func NewGameRoom(gameState *game.Game) *GameRoom {
	roomID := uuid.Must(uuid.NewV4()).String()
	return &GameRoom{
		ID:          roomID,
		Game:        gameState,
		Clients:     make(map[Client]bool),
		BotPlayerID: -1,
	}
}

func (room *GameRoom) notifyStateChange() {
	if room.OnStateChange != nil {
		room.OnStateChange(room)
	}
}

func (room *GameRoom) RefreshTimersLocked() {
	switch room.Game.Phase {
	case game.PhaseCategory:
		room.resetCategoryTimerLocked()
		room.stopRPSTimerLocked()
		room.stopTurnTimerLocked()
	case game.PhaseRPS:
		room.stopCategoryTimerLocked()
		room.resetRPSTimerLocked()
		room.stopTurnTimerLocked()
	case game.PhasePlaying:
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

func (room *GameRoom) SetBotStartDelayLocked(prevPhase game.GamePhase) {
	if room.Game == nil {
		return
	}
	if prevPhase != game.PhasePlaying && room.Game.Phase == game.PhasePlaying {
		if room.BotPlayerID >= 0 && room.Game.CurrentTurn == room.BotPlayerID {
			room.BotStartDelayUntil = time.Now().Add(BotFirstTurnDelay)
		} else {
			room.BotStartDelayUntil = time.Time{}
		}
	}
}

func (room *GameRoom) StopBotActionTimerLocked() {
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
	case game.PhaseCategory:
		_, chosen := room.Game.CategoryChoices[room.BotPlayerID]
		return !chosen
	case game.PhaseRPS:
		_, chosen := room.Game.RPSChoices[room.BotPlayerID]
		return !chosen
	case game.PhasePlaying:
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
	case game.PhaseCategory:
		if _, chosen := room.Game.CategoryChoices[room.BotPlayerID]; chosen {
			return
		}
		category, ok := game.PickBotCategory(room.Game)
		if !ok {
			return
		}
		room.Game.HandleCategorySelection(room.BotPlayerID, category)
	case game.PhaseRPS:
		if _, chosen := room.Game.RPSChoices[room.BotPlayerID]; chosen {
			return
		}
		choices := []string{"rock", "paper", "scissors"}
		choice := choices[rand.Intn(len(choices))]
		room.Game.HandleRPS(room.BotPlayerID, choice)
	case game.PhasePlaying:
		if room.Game.CurrentTurn != room.BotPlayerID {
			return
		}
		room.BotStartDelayUntil = time.Time{}
		room.Game.AutoPlayTurn()
		if room.Game.Phase == game.PhasePlaying && room.Game.CurrentTurn == room.BotPlayerID && room.Game.SharedTile != nil {
			room.BotStartDelayUntil = time.Now().Add(BotConsecutiveTurnDelay)
		}
	}
}

func (room *GameRoom) ScheduleBotAction() {
	if room == nil {
		return
	}
	room.Mutex.Lock()
	if room.BotPlayerID < 0 || room.Game == nil {
		room.StopBotActionTimerLocked()
		room.Mutex.Unlock()
		return
	}
	if !room.shouldBotActLocked() {
		room.StopBotActionTimerLocked()
		room.Mutex.Unlock()
		return
	}
	room.BotActionID++
	id := room.BotActionID
	if room.BotActionTimer != nil {
		room.BotActionTimer.Stop()
	}
	delay := BotActionDelay
	if room.Game.Phase == game.PhasePlaying && room.Game.CurrentTurn == room.BotPlayerID && !room.BotStartDelayUntil.IsZero() {
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
		room.RefreshTimersLocked()
		room.Mutex.Unlock()
		room.notifyStateChange()
		room.ScheduleBotAction()
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
		if id != room.CategoryTimerID || room.Game.Phase != game.PhaseCategory {
			room.Mutex.Unlock()
			return
		}
		changed := room.Game.AutoSelectCategories()
		if changed && len(room.Game.CategoryChoices) == 2 {
			room.Game.Phase = game.PhaseRPS
			room.Game.RPSChoices = make(map[int]string)
			room.Game.StatusMessage = "Categories auto-selected. Play RPS!"
		}
		room.RefreshTimersLocked()
		room.Mutex.Unlock()
		room.notifyStateChange()
		room.ScheduleBotAction()
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
		if id != room.RPSTimerID || room.Game.Phase != game.PhaseRPS {
			room.Mutex.Unlock()
			return
		}
		prevPhase := room.Game.Phase
		if room.Game.AutoSelectRPS() {
			room.Game.ResolveRPS()
		}
		room.SetBotStartDelayLocked(prevPhase)
		room.RefreshTimersLocked()
		room.Mutex.Unlock()
		room.notifyStateChange()
		room.ScheduleBotAction()
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
		if id != room.TurnTimerID || room.Game.Phase != game.PhasePlaying {
			room.Mutex.Unlock()
			return
		}
		room.Game.AutoPlayTurn()
		room.RefreshTimersLocked()
		room.Mutex.Unlock()
		room.notifyStateChange()
		room.ScheduleBotAction()
	})
}
