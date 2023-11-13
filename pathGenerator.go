package main

import (
	"context"
	"github.com/Cladkuu/odo_my_office/state"
	"io"
)

type PathGenerator struct {
	f       io.ReadCloser
	state   *state.State
	closeCH chan struct{}
	buf     []byte
	gen     chan string
	url     []byte
	cancel  context.CancelFunc
}

func NewPathGenerator(
	f io.ReadCloser,
) *PathGenerator {
	return &PathGenerator{
		f:       f,
		state:   state.NewState(),
		closeCH: make(chan struct{}),
		buf:     make([]byte, 32),
		gen:     make(chan string),
		url:     make([]byte, 0, 32),
	}
}

// TODO Доделать
func (p *PathGenerator) Generate(ctx context.Context) {

	go func() {
		// TODO add state
		defer func() {
			close(p.gen)
			p.closeCH <- struct{}{}
		}()

		var n int
		var err error
		var nI int = -1

		for {
			select {
			case <-ctx.Done():
				return
			default:

				for {
					// считываем байты
					n, err = p.f.Read(p.buf)

					// бежим
					for i, b := range p.buf[:n] {
						if b == '\n' {
							nI = i
							break
						}
						p.url = append(p.url, b)
					}

					if nI != -1 {
						p.gen <- string(p.url)
						p.url = p.url[:0]

						for _, b := range p.buf[n+1:] {
							p.url = append(p.url, b)
						}

						nI = -1
					} else if len(p.gen) > 0 && err == io.EOF {
						p.gen <- string(p.url)
						p.url = p.url[:0]
					}

					if err != nil {
						return
					}
				}

			}
		}
	}()

}

func (p *PathGenerator) Close() error {

	<-p.closeCH
	close(p.closeCH)

	p.buf = nil
	p.url = nil

	return nil

}
