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
			Host:         "localhost",
			Port:         3306,
			User:         "root",
			Password:     "admin1234",
			Database:     "example",
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

// 	user.Version++
// 	assert.Equal(t, user.ID, got.ID)
// 	assert.Equal(t, user.Name, got.Name)
// 	assert.EqualValues(t, user.HashedPassword, got.HashedPassword)
// 	assert.Equal(t, user.Email, got.Email)
// }

// func TestUserRepository_Update(t *testing.T) {
// 	db := BuildDBConnectionForTest(t)
// 	ur := infrastructure.NewRepository(db, mediator.Default())
// 	user, err := ur.Get(context.Background(), "f753e5f5-3879-4d62-b45f-83dfc3ba552c")
// 	assert.NoError(t, err)
// 	assert.NotNil(t, user)

// 	defer func() {
// 		user.ChangePassword("test", "123456")
// 		user.Version++
// 		err = ur.Save(context.Background(), user)
// 		assert.NoError(t, err)
// 	}()

// 	err = user.ChangePassword("123455", "test")
// 	assert.NoError(t, err)

// 	err = ur.Save(context.Background(), user)
// 	assert.NoError(t, err)
// }

// func TestQueryUserRepository_CountUserNumber(t *testing.T) {
// 	db := BuildDBConnectionForTest(t)
// 	ur := infrastructure.NewQueryRepository(db)
// 	count, err := ur.CountUserNumber(context.Background(), "test")
// 	assert.NoError(t, err)
// 	assert.Greater(t, count, 1)
// }

// func TestQueryUserRepository_ListUser(t *testing.T) {
// 	db := BuildDBConnectionForTest(t)
// 	ur := infrastructure.NewQueryRepository(db)
// 	users, err := ur.FindUserList(context.Background(), "test", 20, 0)
// 	assert.NoError(t, err)
// 	assert.Greater(t, len(users), 1)
// 	for _, user := range users {
// 		assert.True(t, strings.Contains(user.Name, "test"))
// 	}
// }
