package connectrpc

import (
	"context"
	"errors"

	connect "connectrpc.com/connect"
	"github.com/go-jimu/template/internal/business/user/domain"
)

type internalError struct {
	cause error
}

// Error returns the public message sent over ConnectRPC.
func (internalError) Error() string {
	return "internal server error"
}

// Unwrap preserves the diagnostic cause for the outer logging interceptor.
func (e internalError) Unwrap() error {
	return e.cause
}

func mapError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return connect.NewError(connect.CodeCanceled, err)
	case errors.Is(err, context.DeadlineExceeded):
		return connect.NewError(connect.CodeDeadlineExceeded, err)
	case errors.Is(err, domain.ErrUserNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, domain.ErrInvalidUser), errors.Is(err, domain.ErrInvalidPassword):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, domain.ErrPasswordMismatch):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, domain.ErrConcurrentModification):
		return connect.NewError(connect.CodeAborted, err)
	default:
		return connect.NewError(connect.CodeInternal, internalError{cause: err})
	}
}
