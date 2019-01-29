package request_pool

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"
)

var client *http.Client

func InitClient() {
	client = &http.Client{Transport: &http.Transport{MaxIdleConns: 10, DisableKeepAlives:false}}
	client.Timeout = 2*time.Second
}

const MaxGet = 1

func GetDataFromABC(ctx context.Context, input []string) (ret []string) {
	url := "https://www.google.com/search?q=%s"
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

	ret = make([]string, 0)
	requestHandler := func(req interface{}) interface{} {
		realReq := req.(*Request)
		inputs := input[realReq.Offset : realReq.Offset+realReq.Count]
		inputArrayStr := strings.Join(inputs, ",")
		fetchUrl := fmt.Sprintf(url, inputArrayStr)
		fmt.Println(fetchUrl)
		resp, err := client.Get(fetchUrl)
		if err != nil {
			fmt.Println("err = %+v", err)
			errs := make([]error, 0)
			rets := make([]string, 0)
			for idx := 0; idx < MaxGet; idx++{
				errs = append(errs, errors.New("http err"))
				rets = append(rets, "")
			}
			return &Response{Outputs: rets, Errors: errs}
			// error handler...
		}
		outputs, errs := GetResultFromResp(resp)
		return &Response{Outputs: outputs, Errors: errs}
	}

	responseHandler := func(resp interface{}) {
		realResp := resp.(*Response)
		for idx, err := range realResp.Errors {
			if err != nil {
				// accumulate results
				ret = append(ret, realResp.Outputs[idx])
			}
		}
	}

	requestPool, finishChan := NewWorkerPool(ctx, 10, requestHandler, int64(reqNum), responseHandler)
	defer requestPool.Close()

	offset := 0
	for i := 0; i < reqNum; i++ {
		requestPool.AddRequest(int64(i), &Request{Id: i, Offset: offset, Count: MaxGet})
	}
	<-finishChan
	return
}

func GetResultFromResp(resp *http.Response) (rets []string, errs []error) {
	if resp.StatusCode != http.StatusOK {
		errs = append(errs, errors.New(fmt.Sprintf("http status not ok: %v", resp.StatusCode)))
		rets = append(rets, "")
	}
	body, _ := ioutil.ReadAll(resp.Body)
	rets = append(rets, string(body))
	errs = append(errs, nil)
	return
}

func TestPool(t *testing.T) {
	InitClient()
	input := []string{"a", "b", "c"}
	ctx := context.Background()
	resp := GetDataFromABC(ctx, input)
	for _, r := range resp {
		if len(r) == 0 {
			t.Errorf("resp is nil")
		}
		t.Logf(r)
	}
}
