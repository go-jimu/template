package user

import (
	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/template/internal/driver/rest"
	"github.com/go-jimu/template/internal/user/application"
	"github.com/go-jimu/template/internal/user/infrastructure/persistence"
	"github.com/go-jimu/template/internal/user/infrastructure/port"
	"github.com/jmoiron/sqlx"
)

func Init(m mediator.Mediator, db *sqlx.DB, g rest.ControllerGroup) {
	repo := persistence.NewRepository(db)
	read := persistence.NewQueryRepository(db)
	app := application.NewApplication(m, repo, read)
	controller := port.NewController(app)
	g.With(controller)
}
