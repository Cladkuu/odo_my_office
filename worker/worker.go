package workerPool

import (
	"context"
	"github.com/Cladkuu/odo_my_office/state"
)

// worker
type Worker struct {
	// канал на получение задач
	requests chan Request
	// канал для graceful shutdown
	closeCh chan struct{}
	// состояние
	state *state.State
}

func newWorker() *Worker {
	w := &Worker{
		closeCh:  make(chan struct{}),
		requests: make(chan Request),
		state:    state.NewState(),
	}

	return w
}

func (w *Worker) Run(ctx context.Context) error {
	if err := w.state.Activate(); err != nil {
		return err
	}

	go w.work(ctx)

	return nil
}

func (w *Worker) work(ctx context.Context) {
	defer func() {
		close(w.closeCh)
		_ = w.Close()
	}()

	for {
		select {
		case req, ok := <-w.requests:
			if !ok {
				return
			}

			switch req.RType {
			case RTypeBusiness:
				req.F()
			case RTypeTasksEnded:
				req.F()
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

func (w *Worker) SendTask(req Request) error {
	if err := w.state.IsActive(); err != nil {
		return err
	}

	w.requests <- req
	return nil
}

func (w *Worker) Close() error {
	err := w.state.ShutDown()
	if err != nil {
		return err
	}

	<-w.closeCh
	close(w.requests)

	err = w.state.Close()

	return err
}
