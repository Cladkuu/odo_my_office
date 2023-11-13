package main

import (
	"context"
	"fmt"
	"github.com/Cladkuu/odo_my_office/state"
	workerPool "github.com/Cladkuu/odo_my_office/worker"
	"io"
	"unsafe"
)

type Printer struct {
	worker  workerPool.IWorker
	wr      io.WriteCloser
	state   *state.State
	read    chan HTTPRequesterResponse
	errors  chan error
	closeCH chan struct{}
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

/*func (p *Printer) SendTask(msg string) error {
	if err := p.state.IsActive(); err != nil {
		return err
	}
	p.worker.SendTask(p.createRequest(msg))

	return nil
}*/

func (p *Printer) work(ctx context.Context) {

	defer func() {
		close(p.closeCH)
	}()

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
					return
				}
				continue
			}

			if err := p.state.IsActive(); err != nil {
				return
			}
			p.worker.SendTask(
				func() {
					str := fmt.Sprintf("url:%s, response_size: %d, time_processing: %s", s.path, s.size, s.processed.String())
					_, _ = p.wr.Write(*(*[]byte)(unsafe.Pointer(&str)))
				},
			)
		case e, ok2 = <-p.errors:
			if !ok2 {
				if !ok1 {
					return
				}
				continue
			}
			p.worker.SendTask(
				func() {
					str := e.Error()
					_, _ = p.wr.Write(*(*[]byte)(unsafe.Pointer(&str)))
				},
			)
		}

	}

}

func (p *Printer) Run(ctx context.Context) error {
	if err := p.state.Activate(); err != nil {
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

	_ = p.wr.Close()

	err = p.state.Close()

	return err

}

func (p *Printer) Done() <-chan struct{} {
	return p.closeCH
}
