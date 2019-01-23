package request_pool

type Request struct {
	Id  int64
	Req interface{}
}

type Response struct {
	Id   int64
	Resp interface{}
}