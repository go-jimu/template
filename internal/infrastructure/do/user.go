package do

import (
	"database/sql"
	"time"
)

type UserDO struct {
	ID       string       `db:"id"`
	Name     string       `db:"name"`
	Password sql.RawBytes `db:"password" copier:"HashedPassword"`
	Email    string       `db:"email"`
	Version  int          `db:"version"`
	Deleted  bool         `db:"deleted"`
	CTime    time.Time    `db:"ctime"`
	MTime    time.Time    `db:"mtime"`
}
