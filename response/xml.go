package response

import (
	"encoding/xml"
	"net/http"
)

type XML struct {
	Data interface{}
}

func (r XML) Render(w http.ResponseWriter) error {
	r.WriteContentType(w)
	return xml.NewEncoder(w).Encode(r.Data)
}

func (r XML) WriteContentType(w http.ResponseWriter) {
	writeContentType(w, xmlContentType)
}
