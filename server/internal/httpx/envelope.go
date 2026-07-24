package httpx

type R struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func OK(data any) R { return R{Code: 0, Message: "ok", Data: data} }

func Fail(code int, msg string) R { return R{Code: code, Message: msg, Data: nil} }

type Page struct {
	Total   int64 `json:"total"`
	Records any   `json:"records"`
}
