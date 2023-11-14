package main

import (
	"context"
	"github.com/Cladkuu/odo_my_office/state"
	"io"
)

// стадия, на которой происходит считывание урлов из файла
type PathGenerator struct {
	// reader, откуда считывать данные
	f io.ReadCloser
	// состояние генератора
	state *state.State
	// канал для graceful shutdown.
	closeCH chan struct{}

	// буфер для чтения из f
	buf []byte

	// канал, куда генератор пишет прочитанные данные
	gen chan string

	// слайс для хранения урлов в байтовом представлении
	url []byte

	// функция отмены контекста для генератора
	cancel context.CancelFunc
}

func NewPathGenerator(
	f io.ReadCloser,
) *PathGenerator {
	return &PathGenerator{
		f:       f,
		state:   state.NewState(),
		closeCH: make(chan struct{}, 1),
		buf:     make([]byte, 32),
		gen:     make(chan string),
		url:     make([]byte, 0, 32),
	}
}

func (p *PathGenerator) Generate(ctx context.Context) error {
	if err := p.state.Activate(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	go func() {
		defer func() {
			p.closeCH <- struct{}{}
			p.Close()
		}()

		var n int
		var err error

		for {
			select {
			case <-ctx.Done():
				return
			default:

				// считываем байты
				n, err = p.f.Read(p.buf)

				// бежим
				for _, b := range p.buf[:n] {
					if b == '\n' {
						p.gen <- string(p.url)

						p.url = p.url[:0]
						continue
					}
					p.url = append(p.url, b)
				}

				if err != nil {
					if len(p.url) > 0 {

						p.gen <- string(p.url)
					}
					return
				}

			}
		}
	}()

	return nil
}

func (p *PathGenerator) Close() error {
	err := p.state.ShutDown()
	if err != nil {
		return err
	}

	p.cancel()

	<-p.closeCH
	close(p.closeCH)
	close(p.gen)

	p.buf = nil
	p.url = nil

	err = p.state.Close()

	return err

}
