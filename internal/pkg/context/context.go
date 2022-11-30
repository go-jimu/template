package context

import (
	"context"
	"sync/atomic"
	"time"
)

type (
	Option struct {
		Timeout         string `json:"timeout" yaml:"timeout" toml:"timeout"`
		ShutdownTimeout string `json:"shutdown_timeout" yaml:"shutdown_timeout" toml:"shutdown_timeout"`
	}

	mergedContext struct {
		left       context.Context
		right      context.Context
		doneChan   chan struct{}
		cancelChan chan struct{}
		done       int32
		canceled   int32
		err        error
	}
)

var (
	defaultTimeout  = 30 * time.Second
	shutdownTimeout = 20 * time.Second
)

var parent, cancel = context.WithCancel(context.Background())

func New(opt Option) {
	parent, cancel = context.WithCancel(context.Background())
	duration, err := time.ParseDuration(opt.Timeout)
	if err != nil {
		panic(err)
	}
	defaultTimeout = duration

	duration, err = time.ParseDuration(opt.ShutdownTimeout)
	if err != nil {
		panic(err)
	}
	shutdownTimeout = duration
}

func RootContext() context.Context {
	return parent
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

func KillContextAfterTimeout() {
	KillContextsAfter(shutdownTimeout)
}

// MergeContext
func MergeContext(c1, c2 context.Context) (context.Context, context.CancelFunc) {
	mc := &mergedContext{
		left:       c1,
		right:      c2,
		doneChan:   make(chan struct{}),
		cancelChan: make(chan struct{}),
	}
	go mc.wait()
	return mc, mc.cancel
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
	return mc.doneChan
}

func (mc *mergedContext) cancel() {
	if atomic.CompareAndSwapInt32(&mc.canceled, 0, 1) {
		close(mc.cancelChan)
	}
}

func (mc *mergedContext) wait() {
	defer func() {
		if atomic.CompareAndSwapInt32(&mc.done, 0, 1) {
			close(mc.doneChan)
		}
	}()
	select {
	case <-mc.left.Done():
		mc.err = mc.left.Err()
	case <-mc.right.Done():
		mc.err = mc.right.Err()
	case <-mc.cancelChan:
		mc.err = context.Canceled // 当cancel提前于父context，err为空，不会覆盖；当cancel晚于父context，本函数已经退出，不会覆盖
	}
}

func (mc *mergedContext) Err() error {
	if atomic.LoadInt32(&mc.done) == 1 { // function结束顺序为 wait -> Done -> Err，因此返回结果是可信的
		return mc.err
	}
	return nil
}
