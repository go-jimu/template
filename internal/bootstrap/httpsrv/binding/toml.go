package binding

import (
	"errors"
	"net/http"

	"github.com/pelletier/go-toml/v2"
)

type tomlBinding struct{}

func (tomlBinding) ContentType() []string {
	return []string{ContentTypeToml}
}

func (tomlBinding) Bind(r *http.Request, v any) error {
	if r == nil || r.Body == nil {
		return errors.New("invalid request")
	}
	decoder := toml.NewDecoder(r.Body)
	return decoder.Decode(v)
}
