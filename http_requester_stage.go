package main

import (
	"context"
	"fmt"
	"github.com/Cladkuu/odo_my_office/state"
	workerPool "github.com/Cladkuu/odo_my_office/worker"
	"io/ioutil"
	"net/http"
	"time"
)

type HTTPRequesterResponse struct {
	path      string
	size      int
	processed time.Duration
	err       error
}

// стадия, на которой происходит http-запросы к урлам
type HTTPRequester struct {
	// пул воркеров
	worker workerPool.IWorker
	// канал, куда пишется результат ответа по урлам
	Resp chan HTTPRequesterResponse
	// http-клиент
	client *http.Client
	// состояние
	state *state.State
	// канал, из которого получаем задания
	read chan URLParserResponse
	// канал для отправки ошибок
	error chan error

	// функция отмены контеста
	cancel context.CancelFunc
}

func NewHTTPRequester(
	worker workerPool.IWorker, client *http.Client,
	read chan URLParserResponse,
) *HTTPRequester {
	return &HTTPRequester{
		Resp:   make(chan HTTPRequesterResponse),
		client: client,
		state:  state.NewState(),
		worker: worker,
		read:   read,
		error:  make(chan error),
	}
}

func (h *HTTPRequester) work(ctx context.Context) {
	defer h.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case s, ok := <-h.read:
			if !ok {
				ch := make(chan struct{})
				_ = h.worker.SendTask(
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
			if err := h.state.IsActive(); err != nil {
				return
			}

			if err := h.worker.SendTask(h.createRequest(s)); err != nil {
				return
			}
		}
	}

}

func (h *HTTPRequester) Run(ctx context.Context) error {
	if err := h.state.Activate(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	h.cancel = cancel

	if err := h.worker.Run(ctx); err != nil {
		return err
	}

	go h.work(ctx)

	return nil
}

func (h *HTTPRequester) Close() error {
	err := h.state.ShutDown()
	if err != nil {
		return err
	}

	h.cancel()
	_ = h.worker.Close()
	close(h.Resp)
	close(h.error)

	err = h.state.Close()

	return err
}

func (h *HTTPRequester) createRequest(urlStr URLParserResponse) workerPool.Request {
	return workerPool.Request{
		F: func() {
			now := time.Now()
			resp, err := h.client.Get(urlStr.URL.String())
			if err != nil {

				h.error <- fmt.Errorf("%s failed to Get due to %s", urlStr, err.Error())
				return
			}

			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				h.error <- fmt.Errorf("%s got invalid status code %d", urlStr, resp.StatusCode)
				return
			}

			respMsg := HTTPRequesterResponse{path: urlStr.URL.String()}

			// костыль
			b, _ := ioutil.ReadAll(resp.Body)
			respMsg.size = len(b)
			respMsg.processed = time.Since(now)

			h.Resp <- respMsg

		},
		RType: workerPool.RTypeBusiness,
	}
}
