package state

import (
	"errors"
	"sync/atomic"
)

type State struct {
	s atomic.Int32
}

const (
	initialState = iota
	activeState
	shuttingDownState
	closedState
)

func NewState() *State {
	st := &State{}
	st.s.Store(initialState)
	return st
}

func (s *State) Activate() error {
	if s.s.CompareAndSwap(initialState, activeState) {
		return nil
	}

	return errors.New("not in initial State")
}

func (s *State) IsActive() error {
	if s.s.Load() == activeState {
		return nil
	}

	return errors.New("not in active State")
}

func (s *State) ShutDown() error {
	if s.s.CompareAndSwap(activeState, shuttingDownState) {
		return nil
	}

	return errors.New("not in active State")
}

func (s *State) Close() error {
	if s.s.CompareAndSwap(shuttingDownState, closedState) {
		return nil
	}

	return errors.New("not in shuttingDown State")
}
