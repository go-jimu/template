package user

import (
	"context"
	"fmt"

	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/template/internal/domain/user"
)

type NotificationHandler struct {
}

func (s NotificationHandler) Listening() []mediator.EventKind {
	return []mediator.EventKind{user.EKUserCreated}
}

func (s NotificationHandler) Handle(ctx context.Context, ev mediator.Event) {
	select {
	case <-ctx.Done():
	default:
	}
	fmt.Println(ev)
}
