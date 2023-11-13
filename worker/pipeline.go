package workerPool

import (
	"context"
	"fmt"
	"io"

	"github.com/Cladkuu/odo_my_office/state"
)

type Finalizer interface {
	Done() <-chan struct{}
}

type Generator interface {
	Generate(ctx context.Context)
	io.Closer
}

type Stage interface {
	Run(ctx context.Context) error
	io.Closer
}

type Pipeline struct {
	generator Generator
	finalizer Finalizer
	state     *state.State
	cancel    context.CancelFunc
	stages    []Stage
}

func NewPipeline(generator Generator, finalizer Finalizer, stages ...Stage) *Pipeline {
	p := &Pipeline{
		state:     state.NewState(),
		generator: generator,
	}

	p.stages = stages

	return p
}

func (p *Pipeline) Run(ctx context.Context) error {
	if err := p.state.Activate(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	p.generator.Generate(ctx)

	var err error
	for _, st := range p.stages {
		err = st.Run(ctx)
		if err != nil {
			return err
		}
	}

	select {
	case <-p.finalizer.Done():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Pipeline) Close() error {
	err := p.state.ShutDown()
	if err != nil {
		return err
	}

	p.cancel()

	p.generator.Close()

	for _, stage := range p.stages {
		stage.Close()
	}

	err = p.state.Close()
	fmt.Println("pipe closed")

	return err
}
