package binding

import (
	"errors"
	"net/http"
	"sync"
)

type (
	// Binding describes the interface which needs to be implemented for binding the
	// data present in the request such as JSON request body, query parameters or
	// the form POST.
	Binding interface {
		ContentType() []string
		Bind(*http.Request, any) error
	}

	factory struct {
		bindings map[string]Binding
		mu       sync.RWMutex
	}
)

const (
	ContentTypeBinary            = "application/octet-stream"
	ContentTypeForm              = "application/x-www-form-urlencoded"
	ContentTypeJSON              = "application/json"
	ContentTypeYaml              = "application/x-yaml"
	ContentTypeToml              = "application/toml"
	ContentTypeHTML              = "text/html"
	ContentTypeText              = "text/plain"
	ContentTypeXML               = "application/xml"
	ContentTypeXML2              = "text/xml"
	ContentTypeMultipartPostForm = "multipart/form-data"
	ContentTypeProtoBuf          = "application/x-protobuf"
	ContentTypeMsgPack           = "application/x-msgpack"
	ContentTypeMsgPack2          = "application/msgpack"
)

var defaultFactory *factory

func (f *factory) Registry(b Binding) error {
	if b == nil {
		return errors.New("invalid binding")
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	for _, ct := range b.ContentType() {
		if ct == "" {
			return errors.New("invalid binding")
		}
		f.bindings[ct] = b
	}
	return nil
}

func (f *factory) Get(contentType string) Binding {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.bindings[contentType]
}

func Registry(b Binding) error {
	return defaultFactory.Registry(b)
}

func Get(contentType string) Binding {
	return defaultFactory.Get(contentType)
}

func Default(r *http.Request) Binding {
	if binding := defaultFactory.Get(contentType(r)); binding != nil {
		return binding
	}
	return defaultFactory.Get(ContentTypeForm)
}

func contentType(r *http.Request) string {
	ct := r.Header.Get("Content-Type")
	for i, char := range ct {
		if char == ' ' || char == ';' {
			return ct[:i]
		}
	}
	return ct
}

func init() {
	defaultFactory = &factory{bindings: map[string]Binding{}}

	defaultFactory.Registry(NewJSONBinding())
	defaultFactory.Registry(tomlBinding{})
	defaultFactory.Registry(xmlBinding{})
	defaultFactory.Registry(yamlBinding{})
	defaultFactory.Registry(formBinding{})
	defaultFactory.Registry(formMultipartBinding{})
}
