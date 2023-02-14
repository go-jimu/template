package infrastructure

import (
	"time"
)

type User struct {
	ID       string    `db:"id"`
	Name     string    `db:"name"`
	Password []byte    `db:"password" copier:"HashedPassword"`
	Email    string    `db:"email"`
	Version  int       `db:"version"`
	Deleted  bool      `db:"deleted"`
	CTime    time.Time `db:"ctime"`
	MTime    time.Time `db:"mtime"`
}
