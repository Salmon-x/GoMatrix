package response

import "net/http"

type Render interface {
	Render(w http.ResponseWriter) error
	WriteContentType(w http.ResponseWriter)
}

var jsonContentType = []string{"application/json; charset=utf-8"}
var htmlContentType = []string{"text/html; charset=utf-8"}
var stringContentType = []string{"text/plain; charset=utf-8"}
var xmlContentType = []string{"application/xml; charset=utf-8"}

// 写入ContentType
func writeContentType(w http.ResponseWriter, value []string) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		header["Content-Type"] = value
	}
}
