package binding

import (
	"errors"
	"net/http"

	"github.com/monoculum/formam/v3"
)

type (
	formBinding          struct{}
	formMultipartBinding struct{}
)

const defaultMemory = 32 << 20

var defaultFormOption = &formam.DecoderOptions{TagName: "form"}

func (formBinding) ContentType() []string {
	return []string{ContentTypeForm}
}

func (formBinding) Bind(r *http.Request, v any) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	if err := r.ParseMultipartForm(defaultMemory); err != nil && !errors.Is(err, http.ErrNotMultipart) {
		return err
	}
	return formam.NewDecoder(defaultFormOption).Decode(r.Form, v)
}

func (formMultipartBinding) ContentType() []string {
	return []string{ContentTypeMultipartPostForm}
}

func (formMultipartBinding) Bind(r *http.Request, v any) error {
	if err := r.ParseMultipartForm(defaultMemory); err != nil {
		return err
	}
	return formam.NewDecoder(defaultFormOption).Decode(r.Form, v)
}
