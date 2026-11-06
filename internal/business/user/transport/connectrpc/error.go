package connectrpc

import (
	"context"
	"errors"

	"connectrpc.com/connect/v2"
	"github.com/go-jimu/components/transaction"
	"github.com/go-jimu/template/internal/business/user/domain"
)

// mapError translates an Application error into the Connect error sent to the
// client. Outcomes the client can act on carry the error's message; any other
// error becomes CodeInternal with a fixed message. The original error stays
// the cause in every case, never sent but visible to errors.Is and to the
// interceptor's log.
//
// Transaction outcomes carry a fixed message, since they wrap the driver's
// error. An unknown commit outcome comes first: a commit cut short by a
// cancelled ctx may still have committed, so the client checks before it
// retries.
func mapError(err error) error {
	switch {
	case errors.Is(err, transaction.ErrOutcomeUnknown):
		return connect.NewError(connect.CodeUnknown, "commit outcome unknown; check before retrying").WithCause(err)
	case errors.Is(err, transaction.ErrConflict):
		return connect.NewError(connect.CodeAborted, "transaction conflict; retry").WithCause(err)
	case errors.Is(err, context.Canceled):
		return publicError(connect.CodeCanceled, err)
	case errors.Is(err, context.DeadlineExceeded):
		return publicError(connect.CodeDeadlineExceeded, err)
	case errors.Is(err, domain.ErrUserNotFound):
		return publicError(connect.CodeNotFound, err)
	case errors.Is(err, domain.ErrInvalidUser), errors.Is(err, domain.ErrInvalidPassword):
		return publicError(connect.CodeInvalidArgument, err)
	case errors.Is(err, domain.ErrPasswordMismatch):
		return publicError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, domain.ErrConcurrentModification):
		return publicError(connect.CodeAborted, err)
	default:
		return connect.NewError(connect.CodeInternal, "internal server error").WithCause(err)
	}
}

// publicError sends err's message to the client under code and keeps err as
// the cause, so its fields reach the log.
func publicError(code connect.Code, err error) *connect.Error {
	return connect.NewError(code, err.Error()).WithCause(err)
}
