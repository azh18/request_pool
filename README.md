# Request_Pool

## Introduction

Goroutine is an effective tools of golang which provides high concurrency. Developers often start hundreds of goroutines to handle some I/O requests such as HTTP requests to hide I/O latency. 
To avoid opening too many goroutines, goroutine pool is often used to control the total number of running goroutines, such as <a href=https://github.com/Jeffail/tunny>tunny</a>.
However, you still need to consider asynchonization issues when calling wrapped handler functions and accumulating results in goroutine-unsafe data structure such as slice, map and List.

Request pool is a Golang library for building a goroutine pool for running hundreds of requests concurrently and collecting the results. 
With the help of Request pool, you can easily build a parallel requesting-collecting stream without considering annoying asynchonization issues.
All you need to handle are only implementing a request handler function for sending requests, and a response handler function for processing responses in a synchronous way.
I/O latency is hidden and multiple cores are fully utilized to get high throught.

## Install

Ensure that you install golang >= 1.8, and then run the following scripts:

```
go get github.com/zbw0046/request_pool
```

## Use

We use HTTP requests as the example. Suppose that you need to fetch ```http://abc.com/?offset=0&count=50``` to get 2000 records. However, the maximum limit of each fetch is 50. With the help of request_pool, you can easily write your codes as the following:

### 0. Skeleton

```
package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var client *http.Client

func InitClient() {
	client = &http.Client{Transport: &http.Transport{}}
	client.Timeout = time.Second
}

const MaxGet = 20

func GetDataFromABC(ctx context.Context, input []string) (ret []string) {
	url := "http://abc.com/?inputs=%d"
	reqNum := len(input)/MaxGet + 1
	if len(input)%MaxGet == 0 {
		reqNum = len(input) / MaxGet
	}

	type Request struct {
		Id     int
		Offset int
		Count  int
	}

	type Response struct {
		Outputs []string
		Errors  []error
	}
```

### 1. Implement request handler

```
	requestHandler := func(req interface{}) interface{} {
		realReq := req.(*Request)
		inputs := input[realReq.Offset : realReq.Offset+realReq.Count]
		inputArrayStr := strings.Join(inputs, ",")
		resp, err := client.Get(fmt.Sprintf(url, inputArrayStr))
		if err != nil {
			// error handler...
		}
		outputs, errs := GetResultFromResp(resp)
		return &Response{Outputs: outputs, Errors: errs}
	}
```

### 2. Implement response handler
```
	responseHandler := func(resp interface{}) {
		realResp := resp.(*Response)
		for idx, err := range realResp.Errors {
			if err != nil {
				// accumulate results
				ret = append(ret, realResp.Outputs[idx])
			}
		}
	}
```

### 3. Create your own parallel processing stream
```
	requestPool, finishChan := NewWorkerPool(ctx, 10, requestHandler, int64(reqNum), responseHandler)
	defer requestPool.Close()

	offset := 0
	for i := 0; i < reqNum; i++ {
		requestPool.AddRequest(int64(i), Request{Id: i, Offset: offset, Count: MaxGet})
	}
	<-finishChan
	return
}
```

### 4. Enjoy request pool!







