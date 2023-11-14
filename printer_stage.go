package main

import (
	"context"
	"fmt"
	"github.com/Cladkuu/odo_my_office/state"
	workerPool "github.com/Cladkuu/odo_my_office/worker"
	"io"
)

// стадия, которая печатает результаты в io.Writer
type Printer struct {
	// пул воркеров
	worker workerPool.IWorker
	// поток, в который будет идти запись
	wr io.WriteCloser
	// состояние принтера
	state *state.State
	// канал, из которого приходят обработанные урлы
	read chan HTTPRequesterResponse

	// канал, из которого приходят ошибки
	errors chan error

	// канал для завершения принтера
	closeCH chan struct{}

	cancel context.CancelFunc
}

func NewStdOutPrinter(
	worker workerPool.IWorker, wr io.WriteCloser,
	read chan HTTPRequesterResponse,
	errors chan error,
) *Printer {
	return &Printer{
		wr:      wr,
		state:   state.NewState(),
		worker:  worker,
		read:    read,
		errors:  errors,
		closeCH: make(chan struct{}),
	}
}

func (p *Printer) work(ctx context.Context) {

	defer p.Close()

	var ok1, ok2 bool
	var e error
	var s HTTPRequesterResponse

	for {
		select {
		case <-ctx.Done():
			return
		case s, ok1 = <-p.read:
			if !ok1 {
				if !ok2 {
					p.graceful()
					return
				}
				continue
			}

			if err := p.state.IsActive(); err != nil {
				return
			}
			p.worker.SendTask(
				workerPool.Request{
					F: func() {
						str := fmt.Sprintf("url:%s, response_size: %d, time_processing: %s\n", s.path, s.size, s.processed.String())

						_, _ = p.wr.Write([]byte(str))

					},
					RType: workerPool.RTypeBusiness,
				},
			)
		case e, ok2 = <-p.errors:
			if !ok2 {
				if !ok1 {
					p.graceful()
					return
				}
				continue
			}
			p.worker.SendTask(
				workerPool.Request{
					F: func() {
						_, _ = p.wr.Write(append([]byte(e.Error()), '\n'))

					},
					RType: workerPool.RTypeBusiness,
				},
			)
		}

	}

}

func (p *Printer) graceful() {
	ch := make(chan struct{})
	_ = p.worker.SendTask(
		workerPool.Request{
			F: func() {
				close(ch)
			},
			RType: workerPool.RTypeTasksEnded,
		},
	)
	<-ch
}

func (p *Printer) Run(ctx context.Context) error {
	if err := p.state.Activate(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	if err := p.worker.Run(ctx); err != nil {
		return err
	}

	go p.work(ctx)
	return nil
}

func (p *Printer) Close() error {
	err := p.state.ShutDown()
	if err != nil {
		return err
	}

	p.cancel()
	_ = p.worker.Close()

	_ = p.wr.Close()

	err = p.state.Close()

	close(p.closeCH)

	return err

}

func (p *Printer) Done() <-chan struct{} {
	return p.closeCH
}
