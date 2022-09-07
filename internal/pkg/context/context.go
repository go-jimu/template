package context

import (
	"context"
	"time"
)

type (
	Option struct {
		Timeout         time.Duration
		ShutdownTimeout time.Duration
	}

	mergedContext struct {
		left  context.Context
		right context.Context
	}
)

var (
	parent          context.Context
	cancel          context.CancelFunc
	defaultTimeout  = 30 * time.Second
	shutdownTimeout = 20 * time.Second
)

func New(opt Option) {
	parent, cancel = context.WithCancel(context.Background())
	defaultTimeout = opt.Timeout
	shutdownTimeout = opt.ShutdownTimeout
}

func GenDefaultContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, defaultTimeout)
}

func GenShutdownContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, shutdownTimeout)
}

func GenContextWithTimeout(t time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, t)
}

func KillContextsAfter(t time.Duration) {
	<-time.After(t)
	cancel()
}

func KillContextsImmediately() {
	cancel()
}

// MergeContext TODO: 合并context.Context，解决chi的问题
func MergeContext(c1, c2 context.Context) context.Context {
	return nil
}