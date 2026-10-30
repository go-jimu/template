package application

import "github.com/go-jimu/template/internal/business/user/domain"

// User is the command-side result returned to Transport adapters. It carries
// no Version: under the request-scoped lifecycle a post-Save version is stale.
type User struct {
	ID    string
	Name  string
	Email string
}

// AssembleUserEntity maps existing Domain state to the Application DTO.
func AssembleUserEntity(entity *domain.User) *User {
	if entity == nil {
		return nil
	}
	return &User{ID: entity.ID, Name: entity.Name, Email: entity.Email}
}
