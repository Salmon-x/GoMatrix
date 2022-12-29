package binding

import "net/http"

type Binding interface {
	Name() string
	Bind(*http.Request, interface{}) error
}
