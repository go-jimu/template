package connectrpc

import (
	"encoding/base64"

	userv1 "github.com/go-jimu/template/gen/user/private/v1"
	userpublicv1 "github.com/go-jimu/template/gen/user/public/v1"
	"github.com/go-jimu/template/internal/business/user/application"
	"github.com/go-jimu/template/internal/business/user/application/query"
)

func userToProto(user *application.User) *userv1.User {
	return &userv1.User{Id: user.ID, Name: user.Name, Email: user.Email}
}

func readUserToProto(user query.User) *userv1.User {
	return &userv1.User{Id: user.ID, Name: user.Name, Email: user.Email}
}

// readUserToProfile maps a read model to the public profile: the public
// contract carries only what any client may see.
func readUserToProfile(user query.User) *userpublicv1.Profile {
	return &userpublicv1.Profile{Id: user.ID, Name: user.Name}
}

// encodePageToken and decodePageToken keep the cursor opaque to clients.
func encodePageToken(cursor string) string {
	if cursor == "" {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString([]byte(cursor))
}

func decodePageToken(token string) (string, error) {
	cursor, err := base64.RawURLEncoding.DecodeString(token)
	return string(cursor), err
}
