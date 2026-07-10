package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"github.com/go-jimu/components/sloghelper"
	userv1 "github.com/go-jimu/template/pkg/gen/user/v1"
	"github.com/go-jimu/template/pkg/gen/user/v1/userv1connect"
	"github.com/samber/oops"
)

func main() {
	client := userv1connect.NewUserServiceClient(
		http.DefaultClient,
		"http://localhost:8080",
		connect.WithGRPC(),
	)
	response, err := client.Create(
		context.Background(),
		connect.NewRequest(&userv1.CreateRequest{
			Name:     "template-user",
			Password: "change-me-123",
			Email:    "template@example.com",
		}),
	)
	if err != nil {
		slog.Error("create user failed", sloghelper.Error(oops.Wrap(err)))
		os.Exit(1)
	}
	fmt.Printf("created user: %s\n", response.Msg.GetUser().GetId())
}
