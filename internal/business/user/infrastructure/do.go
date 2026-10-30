package infrastructure

import "github.com/go-jimu/template/internal/pkg/database"

type userDO struct {
	ID        string             `xorm:"id pk"`
	Name      string             `xorm:"name"`
	Password  []byte             `xorm:"password"`
	Email     string             `xorm:"email"`
	Version   int                `xorm:"version"`
	CreatedAt database.Timestamp `xorm:"created_at"`
	UpdatedAt database.Timestamp `xorm:"updated_at"`
	DeletedAt database.Timestamp `xorm:"deleted_at"`
}

func (userDO) TableName() string { return "user_account" }
