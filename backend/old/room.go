//go:build legacy
// +build legacy

package main

import (
	"math/rand"
	"time"

	"github.com/gofrs/uuid"
)

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
