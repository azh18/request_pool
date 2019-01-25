package request_pool

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"time"
)

type WorkerPool struct {
	ctx             context.Context
	reqNum          int64
	waitRequest     chan *Request
	responseQueue   chan *Response
	requestHandler  func(interface{}) interface{}
	responseHandler func(interface{})
	closeChan       chan struct{}
	workers         []*Worker
	response        []*Response
	finishChan      chan struct{}
}

func NewWorkerPool(ctx context.Context, nWorker int, requestHandler func(interface{}) interface{}, reqNum int64, responseHandler func(interface{})) (pool *WorkerPool, finishChan chan struct{}) {
	pool = &WorkerPool{
		ctx:             ctx,
		reqNum:          reqNum,
		waitRequest:     make(chan *Request, reqNum),
		responseQueue:   make(chan *Response, reqNum),
		requestHandler:  requestHandler,
		responseHandler: responseHandler,
		closeChan:       make(chan struct{}),
		finishChan:      make(chan struct{}),
	}
	for i := 0; i < nWorker; i++ {
		worker := NewWorker(pool.waitRequest, pool.responseQueue, pool.requestHandler)
		pool.workers = append(pool.workers, worker)
	}
	go pool.run()
	return pool, pool.finishChan
}

func (p *WorkerPool) AddRequest(reqId int64, payload interface{}) {
	req := &Request{
		Id:  reqId,
		Req: payload,
	}
	p.waitRequest <- req
	return
}

func (p *WorkerPool) Close() {
	p.closeChan <- struct{}{}
}

func (p *WorkerPool) run() {
	defer func() {
		if err := recover(); err != nil {
			debug.PrintStack()
			log.Println(fmt.Sprintf("WorkerPool Panic, err=%+v", err))
		}
	}()
	ctx, cancelFunc := context.WithCancel(p.ctx)
	for _, w := range p.workers {
		go w.Run(ctx)
	}
	go p.runResponseHandler(ctx)
	for {
		select {
		case <-p.closeChan:
			cancelFunc()
			return
		default:
			time.Sleep(LoopSleepDuration)
		}
	}
}

func (p *WorkerPool) runResponseHandler(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			debug.PrintStack()
			log.Println(fmt.Sprintf("runResponseHandler Panic, err=%+v", err))
		}
	}()
	var finishCnt int64 = 0
	for {
		select {
		case resp := <-p.responseQueue:
			p.response = append(p.response, resp)
			p.responseHandler(resp.Resp)
			finishCnt++
			if finishCnt >= p.reqNum {
				p.finishChan <- struct{}{}
			}
		case <-ctx.Done():
			return
		default:
			time.Sleep(LoopSleepDuration)
		}
	}
}
