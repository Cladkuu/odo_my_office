package state

import (
	"errors"
	"sync/atomic"
)

// Состояние объекта
type State struct {
	s atomic.Int32
}

const (
	// иниицализация
	initialState = iota
	// активный
	activeState
	// shutdown
	shuttingDownState
	// закрыт, завершил работу
	closedState
)

func NewState() *State {
	st := &State{}
	st.s.Store(initialState)
	return st
}

// Активирует объект
// Нужно вызвать, когда объект готов к работе, готов принимать задачи
func (s *State) Activate() error {
	if s.s.CompareAndSwap(initialState, activeState) {
		return nil
	}

	return errors.New("not in initial State")
}

// Проверяет, активен ли объект.
// Нужно вызывать, когда объект хочет выполнить бизнес-функцию
func (s *State) IsActive() error {
	if s.s.Load() == activeState {
		return nil
	}

	return errors.New("not in active State")
}

// Перевод в состояние завершения
// Нужно вызывать, когда начинается gracefull shutdown
func (s *State) ShutDown() error {
	if s.s.CompareAndSwap(activeState, shuttingDownState) {
		return nil
	}

	return errors.New("not in active State")
}

// Перевод в завершенное состояние
// Нужно вызвать, когда работа завершена
func (s *State) Close() error {
	if s.s.CompareAndSwap(shuttingDownState, closedState) {
		return nil
	}

	return errors.New("not in shuttingDown State")
}
