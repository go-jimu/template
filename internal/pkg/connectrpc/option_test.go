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
		{name: "random port", option: connectrpc.Option{Addr: "127.0.0.1:0"}},
		{name: "missing address", option: connectrpc.Option{}, wantErr: true},
		{name: "blank address", option: connectrpc.Option{Addr: "  "}, wantErr: true},
		{name: "missing port", option: connectrpc.Option{Addr: "localhost"}, wantErr: true},
		{name: "shutdown delay", option: connectrpc.Option{Addr: ":8080", ShutdownDelay: "5s"}},
		{name: "shutdown delay invalid", option: connectrpc.Option{Addr: ":8080", ShutdownDelay: "soon"}, wantErr: true},
		{name: "shutdown delay negative", option: connectrpc.Option{Addr: ":8080", ShutdownDelay: "-1s"}, wantErr: true},
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
