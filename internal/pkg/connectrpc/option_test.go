package connectrpc_test

import (
	"testing"

	"github.com/go-jimu/template/internal/pkg/connectrpc"
)

func TestOptionValidate(t *testing.T) {
	tests := []struct {
		name    string
		option  connectrpc.Option
		wantErr bool
	}{
		{name: "valid", option: connectrpc.Option{Addr: ":8080"}},
		{name: "missing address", option: connectrpc.Option{}, wantErr: true},
		{name: "missing port", option: connectrpc.Option{Addr: "localhost"}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.option.Validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}
