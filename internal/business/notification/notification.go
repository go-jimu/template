// Package notification demonstrates an Integration Message consuming bounded context.
package notification

import (
	"github.com/go-jimu/components/ddd/message"
	"github.com/go-jimu/template/internal/business/notification/application/command"
	"github.com/go-jimu/template/internal/business/notification/transport/messagesubscriber"
	userintegrationv1 "github.com/go-jimu/template/pkg/gen/user/integration/v1"
	"github.com/samber/oops"
	"go.uber.org/fx"
	"google.golang.org/protobuf/proto"
)

// Module wires the Notification bounded context.
var Module = fx.Module(
	"domain.notification",
	fx.Provide(
		command.NewSendWelcomeNotificationHandler,
		messagesubscriber.NewUserRegisteredSubscriber,
	),
	fx.Invoke(func(
		payloads *message.PayloadRegistry,
		registry message.Subscriber,
		subscriber *messagesubscriber.UserRegisteredSubscriber,
	) error {
		if err := payloads.Register(
			message.KindOf(&userintegrationv1.UserRegisteredV1{}),
			func() proto.Message { return &userintegrationv1.UserRegisteredV1{} },
		); err != nil {
			return oops.With("operation", "message.payload.register").
				Wrap(oops.Wrap(err))
		}
		if err := registry.Subscribe(subscriber); err != nil {
			return oops.With("operation", "message.subscriber.register").
				Wrap(oops.Wrap(err))
		}
		return nil
	}),
)
