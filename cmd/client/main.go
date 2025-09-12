package main

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	userv1 "github.com/go-jimu/template/gen/user/v1"
	"github.com/go-jimu/template/gen/user/v1/userv1connect"
)

func main() {
	client := userv1connect.NewUserAPIClient(http.DefaultClient, "http://localhost:8080", connect.WithGRPC())
	res, err := client.Get(context.Background(), connect.NewRequest(&userv1.GetRequest{Id: "abc"}))
	if err != nil {
		panic(err)
	}
	slog.Info("print result", slog.Any("response", res))
}
