package assembler

import (
	"github.com/go-jimu/template/internal/application/dto"
	"github.com/go-jimu/template/internal/domain/user"
	"github.com/jinzhu/copier"
)

func AssembleDomainUser(entity *user.User) (*dto.User, error) {
	du := new(dto.User)
	if err := copier.Copy(du, entity); err != nil {
		return nil, err
	}
	return du, nil
}
