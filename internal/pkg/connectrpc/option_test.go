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
		{name: "valid", option: connectrpc.Option{Addr: ":8080", PublicAddr: ":8081"}},
		{name: "random ports", option: connectrpc.Option{Addr: "127.0.0.1:0", PublicAddr: "127.0.0.1:0"}},
		{name: "missing address", option: connectrpc.Option{PublicAddr: ":8081"}, wantErr: true},
		{name: "missing public address", option: connectrpc.Option{Addr: ":8080"}, wantErr: true},
		{name: "missing port", option: connectrpc.Option{Addr: "localhost", PublicAddr: ":8081"}, wantErr: true},
		// One port would expose the private contracts wherever the public
		// ones are reachable.
		{name: "same address", option: connectrpc.Option{Addr: ":8080", PublicAddr: ":8080"}, wantErr: true},
		{name: "same port, wildcard", option: connectrpc.Option{Addr: ":8080", PublicAddr: "127.0.0.1:8080"}, wantErr: true},
		{name: "shutdown delay", option: connectrpc.Option{Addr: ":8080", PublicAddr: ":8081", ShutdownDelay: "5s"}},
		{name: "shutdown delay invalid", option: connectrpc.Option{Addr: ":8080", PublicAddr: ":8081", ShutdownDelay: "soon"}, wantErr: true},
		{name: "shutdown delay negative", option: connectrpc.Option{Addr: ":8080", PublicAddr: ":8081", ShutdownDelay: "-1s"}, wantErr: true},
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
