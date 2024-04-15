package binding

import (
	"encoding/json"
	"errors"
	"net/http"
)

type (
	JSONBinding struct {
		useNumber             bool
		disallowUnknownFields bool
	}

	JSONBindingOption func(*JSONBinding)
)

func NewJSONBinding(opts ...JSONBindingOption) Binding {
	binding := &JSONBinding{}
	binding.Apply(opts...)
	return binding
}

func (binding *JSONBinding) ContentType() []string {
	return []string{ContentTypeJSON}
}

func (binding *JSONBinding) Bind(r *http.Request, v any) error {
	if r == nil || r.Body == nil {
		return errors.New("invalid request")
	}

	decoder := json.NewDecoder(r.Body)
	if binding.useNumber {
		decoder.UseNumber()
	}
	if binding.disallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	return decoder.Decode(v)
}

func (binding *JSONBinding) Apply(opts ...JSONBindingOption) {
	for _, opt := range opts {
		opt(binding)
	}
}

func WithUseNumber(enable bool) JSONBindingOption {
	return func(binding *JSONBinding) {
		binding.useNumber = enable
	}
}

func WithDisallowUnknownFields(enable bool) JSONBindingOption {
	return func(binding *JSONBinding) {
		binding.disallowUnknownFields = enable
	}
}
