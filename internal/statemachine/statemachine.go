package statemachine


import (
	"fmt"
	"sync"
)

type State string

const (
	StateIdle     State = "idle"
	StateReady    State = "ready"
	StateRound1   State = "round_1"
	StateBreak1   State = "break_1"
	StateRound2   State = "round_2"
	StateBreak2   State = "break_2"
	StateRound3   State = "round_3"
	StateFinished State = "finished"
)

type Event string

const (
	EventStart      Event = "start"
	EventRoundEnd   Event = "round_end"
	EventBreakEnd   Event = "break_end"
	EventFinish     Event = "finish"
	EventReset      Event = "reset"
)

var transitions = map[State]map[Event]State{
	StateIdle:     {EventStart: StateReady},
	StateReady:    {EventStart: StateRound1},
	StateRound1:   {EventRoundEnd: StateBreak1, EventFinish: StateFinished},
	StateBreak1:   {EventBreakEnd: StateRound2},
	StateRound2:   {EventRoundEnd: StateBreak2, EventFinish: StateFinished},
	StateBreak2:   {EventBreakEnd: StateRound3},
	StateRound3:   {EventRoundEnd: StateFinished, EventFinish: StateFinished},
	StateFinished: {EventReset: StateIdle},
}

type StateMachine struct {
	mu      sync.Mutex
	current State
	MatchID string
}

func New(matchID string) *StateMachine {
	return &StateMachine{
		current: StateIdle,
		MatchID: matchID,
	}
}

func NewWithState(matchID string, state State) *StateMachine {
	return &StateMachine{
		current: state,
		MatchID: matchID,
	}
}

func (sm *StateMachine) Current() State {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.current
}

func (sm *StateMachine) Transition(event Event) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	next, ok := transitions[sm.current][event]
	if !ok {
		return fmt.Errorf("invalid transition: %s -[%s]-> ?", sm.current, event)
	}

	sm.current = next
	return nil
}

func (sm *StateMachine) IsFinished() bool {
	return sm.Current() == StateFinished
}

func (sm *StateMachine) IsInRound() bool {
	s := sm.Current()
	return s == StateRound1 || s == StateRound2 || s == StateRound3
}

func (sm *StateMachine) CurrentRoundNumber() int {
	switch sm.Current() {
	case StateRound1, StateBreak1:
		return 1
	case StateRound2, StateBreak2:
		return 2
	case StateRound3, StateFinished:
		return 3
	default:
		return 0
	}
}