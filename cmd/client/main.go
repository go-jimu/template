package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/go-jimu/components/sloghelper"
	userv1 "github.com/go-jimu/template/gen/user/private/v1"
	"github.com/go-jimu/template/gen/user/private/v1/userv1connect"
	userpublicv1 "github.com/go-jimu/template/gen/user/public/v1"
	"github.com/go-jimu/template/gen/user/public/v1/userpublicv1connect"
	"github.com/samber/oops"
)

func main() {
	client := userv1connect.NewUserServiceClient(connect.NewClient(connecthttp.NewTransport(
		http.DefaultClient,
		"http://localhost:8080",
		connecthttp.WithGRPC(),
	)))
	response, err := client.Create(
		context.Background(),
		&userv1.CreateRequest{
			Name:     "template-user",
			Password: "change-me-123",
			Email:    "template@example.com",
		},
	)
	if err != nil {
		slog.Error("create user failed", sloghelper.Error(oops.Wrap(err)))
		os.Exit(1)
	}
	userID := response.GetUser().GetId()
	fmt.Printf("created user: %s\n", userID)

	// A web or app client uses the public contract. GetProfile has no side
	// effects, so it goes out as a cacheable HTTP GET.
	profiles := userpublicv1connect.NewProfileServiceClient(connect.NewClient(connecthttp.NewTransport(
		http.DefaultClient,
		"http://localhost:8080",
		connecthttp.WithHTTPGet(),
	)))
	profile, err := profiles.GetProfile(
		context.Background(),
		&userpublicv1.GetProfileRequest{Id: userID},
	)
	if err != nil {
		slog.Error("get profile failed", sloghelper.Error(oops.Wrap(err)))
		os.Exit(1)
	}
	fmt.Printf("public profile: %s\n", profile.GetProfile().GetName())
}
