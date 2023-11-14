package main

import (
	"context"
	"github.com/Cladkuu/odo_my_office/state"
	"sync"
)

// стадия, на которой происходит работа с ошибками, возникшими
// во время исполнения
// fan-in pattern.
// Перенаправляем поток сообщения из нескольких каналов read в один Write
type ErrorManager struct {
	// Канал, в который идет запись
	Write chan error

	// для контроля над несколькими горутинами
	wg *sync.WaitGroup

	// состояние
	state *state.State
	// каналы, из которых читает менеджер
	read []<-chan error

	// функция отмены контекста
	cancel context.CancelFunc
}

func NewErrorManager(
	read ...<-chan error,
) *ErrorManager {
	return &ErrorManager{
		Write: make(chan error),
		state: state.NewState(),
		read:  read,
		wg:    new(sync.WaitGroup),
	}

}

func (h *ErrorManager) work(ctx context.Context) {

	for _, r := range h.read {
		h.wg.Add(1)
		go func(r <-chan error) {

			defer h.wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-r:
					if !ok {
						return
					}

					h.Write <- msg

				}
			}

		}(r)
	}

	h.wg.Wait()
	h.Close()

}

func (h *ErrorManager) Run(ctx context.Context) error {
	if err := h.state.Activate(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	h.cancel = cancel

	go h.work(ctx)

	return nil
}

func (h *ErrorManager) Close() error {
	err := h.state.ShutDown()
	if err != nil {
		return err
	}

	h.cancel()
	h.wg.Wait()

	err = h.state.Close()
	close(h.Write)

	return err
}
