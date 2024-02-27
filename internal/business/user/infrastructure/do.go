package infrastructure

import (
	"time"

	"github.com/uptrace/bun"
)

type User struct {
	bun.BaseModel `bun:"table:user,alias:u"`
	ID            string       `bun:"id,pk"`
	Name          string       `bun:"name"`
	Password      []byte       `bun:"password" copier:"HashedPassword"`
	Email         string       `bun:"email"`
	Version       int          `bun:"version"`
	Deleted       bool         `bun:"deleted"`
	CTime         bun.NullTime `bun:"ctime"`
	MTime         time.Time    `bun:"mtime"`
}
