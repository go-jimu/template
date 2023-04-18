package binding

import (
	"encoding/xml"
	"errors"
	"net/http"
)

type xmlBinding struct{}

func (xb xmlBinding) ContentType() []string {
	return []string{ContentTypeXML, ContentTypeXML2}
}

func (xb xmlBinding) Bind(r *http.Request, v any) error {
	if r == nil || r.Body == nil {
		return errors.New("invalid request")
	}
	decoder := xml.NewDecoder(r.Body)
	return decoder.Decode(v)
}
