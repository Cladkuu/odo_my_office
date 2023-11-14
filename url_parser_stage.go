package main

import (
	"context"
	"fmt"
	"github.com/Cladkuu/odo_my_office/state"
	workerPool "github.com/Cladkuu/odo_my_office/worker"
	"net/url"
)

type URLParserResponse struct {
	URL *url.URL
}

// стадия, проверяющая валидность урлов
type URLParser struct {
	// пул воркеров
	worker workerPool.IWorker
	// канал, куда будут отправляться валидные урлы
	Resp chan URLParserResponse

	// канал, куда будут отправляться не валидные урлы
	error chan error

	// канал, из которого парсер получает задания
	read chan string

	// состояние парсера
	state *state.State

	// функция по отмене контеста парсера
	cancel context.CancelFunc
}

func NewURLParser(
	worker workerPool.IWorker,
	read chan string,
) *URLParser {
	return &URLParser{
		Resp:   make(chan URLParserResponse),
		state:  state.NewState(),
		worker: worker,
		read:   read,
		error:  make(chan error),
	}
}

func (u *URLParser) Run(ctx context.Context) error {
	if err := u.state.Activate(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	u.cancel = cancel

	if err := u.worker.Run(ctx); err != nil {
		return err
	}

	go u.work(ctx)

	return nil
}

func (u *URLParser) work(ctx context.Context) {

	defer u.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case s, ok := <-u.read:
			if !ok {
				ch := make(chan struct{})
				_ = u.worker.SendTask(
					workerPool.Request{
						F: func() {
							close(ch)
						},
						RType: workerPool.RTypeTasksEnded,
					},
				)
				<-ch

				return
			}
			if err := u.state.IsActive(); err != nil {
				return
			}

			err := u.worker.SendTask(u.createRequest(s))
			if err != nil {
				return
			}

		}
	}

}

func (u *URLParser) Close() error {
	err := u.state.ShutDown()
	if err != nil {
		return err
	}

	u.cancel()
	_ = u.worker.Close()
	close(u.Resp)
	close(u.error)

	err = u.state.Close()

	return err
}

func (u *URLParser) createRequest(urlStr string) workerPool.Request {
	return workerPool.Request{
		F: func() {
			parsedURL, err := url.ParseRequestURI(urlStr)
			if err != nil {

				u.error <- fmt.Errorf("%s failed parsing due to: %s", urlStr, err.Error())
			} else {
				u.Resp <- URLParserResponse{URL: parsedURL}
			}
		},
		RType: workerPool.RTypeBusiness,
	}
}
