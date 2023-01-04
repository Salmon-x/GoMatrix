package binding

import "net/http"

// TODO 解析结构体与参数匹配

type Binding interface {
	Name() string
	Bind(*http.Request, interface{}) error
}
