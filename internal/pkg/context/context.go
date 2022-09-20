package context

import (
	"context"
	"sync/atomic"
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
		ch    chan struct{}
		done  int32
		err   error
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
	mc := &mergedContext{
		left:  c1,
		right: c2,
		ch:    make(chan struct{}),
	}
	go mc.wait()
	return mc
}

func (mc *mergedContext) Value(key any) any {
	val := mc.left.Value(key)
	if val == nil {
		return mc.right.Value(key)
	}
	return val
}

func (mc *mergedContext) Deadline() (time.Time, bool) {
	t1, o1 := mc.left.Deadline()
	t2, o2 := mc.right.Deadline()

	switch {
	case !o1 && !o2: // no set
		return t1, o1

	case o1 && o2:
		if t1.Before(t2) {
			return t1, true
		}
		return t2, true

	default:
		if o1 {
			return t1, o1
		}
		return t2, o2
	}
}

func (mc *mergedContext) Done() <-chan struct{} {
	return mc.ch
}

func (mc *mergedContext) wait() {
	defer func() {
		if atomic.CompareAndSwapInt32(&mc.done, 0, 1) {
			close(mc.ch)
		}
	}()
	select {
	case <-mc.left.Done():
		mc.err = mc.left.Err()
	case <-mc.right.Done():
		mc.err = mc.right.Err()
	}
}

func (mc *mergedContext) Err() error {
	if atomic.LoadInt32(&mc.done) == 1 {
		return mc.err
	}
	return nil
}
