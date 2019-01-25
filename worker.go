package request_pool

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"time"
)

const LoopSleepDuration = 500 * time.Microsecond

type Worker struct {
	poolWaitRequest   chan *Request
	poolResponseQueue chan *Response
	handler           func(interface{}) interface{}
}

func NewWorker(waitRequestChan chan *Request, responseChan chan *Response, handler func(interface{}) interface{}) *Worker {
	return &Worker{
		poolWaitRequest:   waitRequestChan,
		poolResponseQueue: responseChan,
		handler:           handler,
	}
}

func (w *Worker) Run(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			debug.PrintStack()
			log.Println(fmt.Sprintf("FlushBugDescriptionInDB Panic, err=%+v", err))
		}
	}()
	for {
		select {
		case request := <-w.poolWaitRequest:
			resp := w.handler(request.Req)
			w.poolResponseQueue <- &Response{request.Id, resp}
		case <-ctx.Done():
			return
		default:
			time.Sleep(LoopSleepDuration)
		}
	}
}
