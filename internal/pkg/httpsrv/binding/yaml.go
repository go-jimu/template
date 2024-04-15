package binding

import (
	"errors"
	"net/http"

	"gopkg.in/yaml.v3"
)

type yamlBinding struct{}

func (yamlBinding) ContentType() []string {
	return []string{ContentTypeYaml}
}

func (yamlBinding) Bind(r *http.Request, v any) error {
	if r == nil || r.Body == nil {
		return errors.New("invalid request")
	}
	decoder := yaml.NewDecoder(r.Body)
	return decoder.Decode(v)
}
