package infrastructure

import (
	"database/sql"
	"time"
)

type UserDO struct {
	ID        string       `xorm:"id"`
	Name      string       `xorm:"name"`
	Password  []byte       `xorm:"password" copier:"HashedPassword"`
	Email     string       `xorm:"email"`
	Version   int          `xorm:"version"`
	CreatedAt time.Time    `xorm:"created_at <-"`
	UpdatedAt time.Time    `xorm:"updated_at <-"`
	DeletedAt sql.NullTime `xorm:"deleted_at <-"`
}

func (u UserDO) TableName() string {
	return "user"
}
