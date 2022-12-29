package binding

import (
	"fmt"
	"io"
	"net/http"
)

type jsonBinding struct{}

func (jsonBinding) Name() string {
	return "json"
}

func (jsonBinding) Bind(req *http.Request, obj interface{}) error {
	if req == nil || req.Body == nil {
		return fmt.Errorf("invalid request")
	}
	return decodeJSON(req.Body, obj)
}

func decodeJSON(r io.Reader, obj interface{}) error {
	return nil
	//decoder := json.NewDecoder(r)
	//if EnableDecoderUseNumber {
	//	decoder.UseNumber()
	//}
	//if EnableDecoderDisallowUnknownFields {
	//	decoder.DisallowUnknownFields()
	//}
	//if err := decoder.Decode(obj); err != nil {
	//	return err
	//}
	//return validate(obj)
}
