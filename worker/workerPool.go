package workerPool

import (
	"context"
	"io"
	"sync"

	"github.com/Cladkuu/odo_my_office/state"
)

// Интерфейс Воркера и Воркер пула
type IWorker interface {
	io.Closer
	SendTask(req Request) error
	Run(ctx context.Context) error
}

// структура, отвечающая за работу с воркерами
type workers struct {
	// текущее кол-во воркеров
	workerCount int
	// слайс воркеров
	workers []IWorker
	// индекс воркера, которому была оправлена последняя задача
	// нужен для round-robin по выбору воркера
	workerIndex int
	mutex       *sync.Mutex
}

func (w *workers) sendTaskToAll(req Request) error {
	var err error
	switch req.RType {
	case RTypeTasksEnded:
		ch := make(chan struct{})
		defer close(ch)

		r := Request{RType: req.RType}
		r.F = func() {
			ch <- struct{}{}
		}

		for _, work := range w.workers {
			err = work.SendTask(r)
			if err != nil {
				return err
			}

			<-ch
		}

	}

	return nil
}

func (w *workers) Close() error {
	for _, work := range w.workers {
		work.Close()
	}

	return nil
}

func (w *workers) Run(ctx context.Context) error {
	var err error
	for _, work := range w.workers {
		err = work.Run(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (w *workers) GetWorker() IWorker {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.workerIndex++
	if w.workerIndex == len(w.workers) {
		w.workerIndex = 0
	}

	return w.workers[w.workerIndex]
}

func newWorkers(count int) *workers {
	w := &workers{
		mutex:       &sync.Mutex{},
		workerCount: count,
	}

	if w.workerCount <= 0 {
		w.workerCount = 5
	}

	w.workers = make([]IWorker, 0, w.workerCount)
	for i := 0; i < w.workerCount; i++ {
		w.workers = append(
			w.workers, newWorker(),
		)
	}

	return w
}

// воркер пул
type WorkerPool struct {
	// канал задач
	requests <-chan Request
	// вокеры
	workers *workers
	// отмена контеста
	cancel context.CancelFunc
	// состояние
	state *state.State
}

func NewWorkerPool(count int) *WorkerPool {
	return &WorkerPool{
		workers:  newWorkers(count),
		state:    state.NewState(),
		requests: make(chan Request),
	}
}

func (w *WorkerPool) Run(ctx context.Context) error {
	err := w.state.Activate()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel

	err = w.workers.Run(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (w *WorkerPool) SendTask(req Request) error {
	if err := w.state.IsActive(); err != nil {
		return err
	}

	var err error

	// TODO переделать в будущем
	switch req.RType {
	case RTypeBusiness:
		err = w.workers.GetWorker().SendTask(req)

	case RTypeTasksEnded:
		err = w.workers.sendTaskToAll(req)
		if err != nil {
			return err
		}
		_ = w.Close()
		req.F()
	}

	return nil
}

func (w *WorkerPool) Close() error {
	err := w.state.ShutDown()
	if err != nil {
		return err
	}

	w.cancel()

	w.workers.Close()

	err = w.state.Close()

	return err
}
