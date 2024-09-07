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
	CreatedAt time.Time    `xorm:"ctime,<-"`
	UpdatedAt time.Time    `xorm:"mtime,<-"`
	DeletedAt sql.NullTime `xorm:"deleted,<-"`
}

func (u UserDO) TableName() string {
	return "user"
}
