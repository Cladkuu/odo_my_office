// Реализация пайплайна

package workerPool

import (
	"context"
	"io"

	"github.com/Cladkuu/odo_my_office/state"
)

// Интерфейс финальной стадии.
// Нужен, чтобы понять, что все стадии завершились
type Finalizer interface {
	Done() <-chan struct{}
}

// Интерфейс генератора данных
type Generator interface {
	Generate(ctx context.Context) error
	io.Closer
}

// Интерфейс стадии
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
	return &Pipeline{
		state:     state.NewState(),
		generator: generator,
		finalizer: finalizer,
		stages:    stages,
	}

}

// Запускаем пайплайн
func (p *Pipeline) Run(ctx context.Context) error {
	if err := p.state.Activate(); err != nil {
		return err
	}

	defer p.Close()

	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	// запускаем генератор.
	// Генерация должна происходить в отлельной горутине
	err := p.generator.Generate(ctx)
	if err != nil {
		return err
	}

	// запускаем стадии
	// работа стадии должна происходить в отдельной горутине
	for _, st := range p.stages {
		err = st.Run(ctx)
		if err != nil {
			return err
		}
	}

	// Ждем результат
	// Либо все стадии завершены
	// Либо завершение по таймауту
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

	return err
}
