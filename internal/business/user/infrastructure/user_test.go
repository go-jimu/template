package infrastructure_test

import (
	"log/slog"
	"testing"

	"github.com/go-jimu/template/internal/pkg/database"
	"go.uber.org/fx/fxtest"
	"xorm.io/xorm"
)

func BuildDBConnectionForTest(tb testing.TB) *xorm.Engine {
	lc := fxtest.NewLifecycle(tb)
	db, err := database.NewMySQLDriver(
		lc,
		database.Option{
			Host:         "127.0.0.1",
			Port:         3306,
			User:         "root",
			Password:     "jimu",
			Database:     "jimu",
			MaxOpenConns: 5,
			MaxIdleConns: 3,
			MaxIdleTime:  "10m",
		},
		slog.Default())
	if err != nil {
		tb.FailNow()
	}
	return db
}

// func TestUserRepository_Get(t *testing.T) {
// 	db := BuildDBConnectionForTest(t)
// 	ur := infrastructure.NewRepository(db, mediator.Default())
// 	user, err := ur.Get(context.Background(), "1")
// 	assert.NoError(t, err)
// 	assert.NotNil(t, user)
// }

// func TestUserRepository_Create(t *testing.T) {
// 	db := BuildDBConnectionForTest(t)
// 	ur := infrastructure.NewRepository(db, mediator.Default())
// 	user, err := domain.NewUser(
// 		"test",
// 		"123456",
// 		"test@example.com",
// 	)

// 	assert.NoError(t, err)
// 	assert.NotNil(t, user)

// 	err = ur.Save(context.Background(), user)
// 	assert.NoError(t, err)

// 	got, err := ur.Get(context.Background(), user.ID)
// 	assert.NoError(t, err)
// 	assert.NotNil(t, got)
// 	t.Log(got.CreatedAt, got.UpdatedAt, got.Deleted)

// 	got.Deleted = true
// 	err = ur.Save(context.Background(), got)
// 	assert.NoError(t, err)

// 	got, err = ur.Get(context.Background(), user.ID)
// 	assert.NotNil(t, err)
// 	// user.Version++
// 	// assert.Equal(t, user.ID, got.ID)
// 	// assert.Equal(t, user.Name, got.Name)
// 	// assert.EqualValues(t, user.HashedPassword, got.HashedPassword)
// 	// assert.Equal(t, user.Email, got.Email)
// }
