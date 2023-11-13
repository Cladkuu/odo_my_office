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

type URLParser struct {
	worker workerPool.IWorker
	Resp   chan URLParserResponse
	error  chan<- error
	read   chan string
	state  *state.State
}

func NewURLParser(
	worker workerPool.IWorker,
	read chan string,
	error chan<- error,
) *URLParser {
	return &URLParser{
		Resp:   make(chan URLParserResponse),
		state:  state.NewState(),
		worker: worker,
		read:   read,
		error:  error,
	}
}

/*func (u *URLParser) SendTask(url string) error {
	if err := u.state.IsActive(); err != nil {
		return err
	}

	return u.worker.SendTask(u.createRequest(url))
}*/

func (u *URLParser) Run(ctx context.Context) error {
	if err := u.state.Activate(); err != nil {
		return err
	}

	if err := u.worker.Run(ctx); err != nil {
		return err
	}

	go u.work(ctx)

	return nil
}

func (u *URLParser) work(ctx context.Context) {

	for {
		select {
		case <-ctx.Done():
			return
		case s, ok := <-u.read:
			if !ok {
				return
			}
			if err := u.state.IsActive(); err != nil {
				return
			}

			u.worker.SendTask(u.createRequest(s))

		}
	}

}

func (u *URLParser) Close() error {
	err := u.state.ShutDown()
	if err != nil {
		return err
	}

	_ = u.worker.Close()
	close(u.Resp)

	err = u.state.Close()

	return err
}

func (u *URLParser) createRequest(urlStr string) func() {
	return func() {
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			u.error <- fmt.Errorf("%s failed parsing due to: %s", err.Error())
		} else {
			u.Resp <- URLParserResponse{URL: parsedURL}
		}

	}
}
