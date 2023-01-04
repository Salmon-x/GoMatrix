package response

import (
	"fmt"
	"github.com/Salmon-x/GoMatrix/utils"
	"net/http"
)

type String struct {
	Format string
	Value  []interface{}
}

func (s String) Render(w http.ResponseWriter) (err error) {
	return WriteStr(w, s.Format, s.Value)
}

func WriteStr(w http.ResponseWriter, format string, data []interface{}) (err error) {
	writeContentType(w, stringContentType)
	if len(data) > 0 {
		_, err = fmt.Fprintf(w, format, data...)
		return
	}
	_, err = w.Write(utils.StringToBytes(format))
	return
}

func (s String) WriteContentType(w http.ResponseWriter) {
	writeContentType(w, stringContentType)
}
