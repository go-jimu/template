package infrastructure

import (
	"github.com/go-jimu/template/internal/pkg/database"
)

type UserDO struct {
	ID        string                 `xorm:"id pk"`
	Name      string                 `xorm:"name"`
	Password  []byte                 `xorm:"password" copier:"HashedPassword"`
	Email     string                 `xorm:"email"`
	Version   int                    `xorm:"version"`
	CreatedAt database.Timestamp     `xorm:"created_at"`
	UpdatedAt database.Timestamp     `xorm:"updated_at"`
	DeletedAt database.NullTimestamp `xorm:"deleted_at"`
}

func (u UserDO) TableName() string {
	return "user"
}
