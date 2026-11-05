package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"github.com/go-jimu/components/sloghelper"
	userv1 "github.com/go-jimu/template/gen/user/private/v1"
	"github.com/go-jimu/template/gen/user/private/v1/userv1connect"
	userpublicv1 "github.com/go-jimu/template/gen/user/public/v1"
	"github.com/go-jimu/template/gen/user/public/v1/userpublicv1connect"
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
	userID := response.Msg.GetUser().GetId()
	fmt.Printf("created user: %s\n", userID)

	// A web or app client uses the public contract. GetProfile has no side
	// effects, so it goes out as a cacheable HTTP GET.
	profiles := userpublicv1connect.NewProfileServiceClient(
		http.DefaultClient,
		"http://localhost:8081", // the public port
		connect.WithHTTPGet(),
	)
	profile, err := profiles.GetProfile(
		context.Background(),
		connect.NewRequest(&userpublicv1.GetProfileRequest{Id: userID}),
	)
	if err != nil {
		slog.Error("get profile failed", sloghelper.Error(oops.Wrap(err)))
		os.Exit(1)
	}
	fmt.Printf("public profile: %s\n", profile.Msg.GetProfile().GetName())
}
