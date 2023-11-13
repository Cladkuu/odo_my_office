package main

import (
	"context"
	"fmt"
	"github.com/Cladkuu/odo_my_office/state"
	workerPool "github.com/Cladkuu/odo_my_office/worker"
	"net/http"
	"strconv"
	"time"
)

type HTTPRequesterResponse struct {
	path      string
	size      int
	processed time.Duration
	err       error
}

type HTTPRequester struct {
	worker workerPool.IWorker
	Resp   chan HTTPRequesterResponse
	client *http.Client
	state  *state.State
	read   chan URLParserResponse
	error  chan<- error
}

func NewHTTPRequester(
	worker workerPool.IWorker, client *http.Client,
	read chan URLParserResponse,
	error chan<- error,
) *HTTPRequester {
	return &HTTPRequester{
		Resp:   make(chan HTTPRequesterResponse),
		client: client,
		state:  state.NewState(),
		worker: worker,
		read:   read,
		error:  error,
	}
}

/*func (h *HTTPRequester) SendTask(url string) error {
	if err := h.state.IsActive(); err != nil {
		return err
	}
	return h.worker.SendTask(h.createRequest(url))
}*/

func (h *HTTPRequester) work(ctx context.Context) {

	for {
		select {
		case <-ctx.Done():
			return
		case s, ok := <-h.read:
			if !ok {
				return
			}
			if err := h.state.IsActive(); err != nil {
				return
			}

			h.worker.SendTask(h.createRequest(s))
		}
	}

}

func (h *HTTPRequester) Run(ctx context.Context) error {
	if err := h.state.Activate(); err != nil {
		return err
	}
	return h.worker.Run(ctx)
}

func (h *HTTPRequester) Close() error {
	err := h.state.ShutDown()
	if err != nil {
		return err
	}

	_ = h.worker.Close()
	close(h.Resp)

	err = h.state.Close()

	return err
}

func (h *HTTPRequester) createRequest(urlStr URLParserResponse) func() {
	return func() {
		now := time.Now()
		resp, err := h.client.Get(urlStr.URL.String())
		if err != nil {
			h.error <- fmt.Errorf("%s failed to Get due to %s", urlStr, err.Error())
			return
		}

		respMsg := HTTPRequesterResponse{}

		respMsg.size /*respMsg.err*/, _ = strconv.Atoi(resp.Header.Get("Content-Length"))
		/*respMsg.err = err*/

		respMsg.processed = time.Since(now)

		h.Resp <- respMsg

	}
}
