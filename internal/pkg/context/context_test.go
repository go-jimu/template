package context

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDeadline(t *testing.T) {
	ctx1, _ := context.WithTimeout(context.Background(), 1*time.Second)
	ctx2, _ := context.WithTimeout(context.Background(), 2*time.Second)
	now := time.Now()
	mc, _ := MergeContext(ctx1, ctx2)
	d, ok := mc.Deadline()
	duration := d.Sub(now)
	assert.True(t, duration <= 1*time.Second)
	assert.True(t, ok)

	mc, _ = MergeContext(ctx2, ctx1)
	d, ok = mc.Deadline()
	duration = d.Sub(now)
	assert.True(t, duration <= 1*time.Second)
	assert.True(t, ok)

	mc, _ = MergeContext(ctx2, context.Background())
	d, ok = mc.Deadline()
	duration = d.Sub(now)
	assert.True(t, duration <= 2*time.Second)
	assert.True(t, ok)

	ctx1 = context.Background()
	ctx2 = context.Background()
	mc, _ = MergeContext(ctx1, ctx2)
	_, ok = mc.Deadline()
	assert.False(t, ok)
}

func TestDone(t *testing.T) {
	c1, ca1 := context.WithCancel(context.Background())
	c2, ca2 := context.WithCancel(context.Background())
	mc, _ := MergeContext(c1, c2)

	go func() {
		time.Sleep(1 * time.Second)
		cancels := []context.CancelFunc{ca1, ca2}
		n := rand.Int() % len(cancels)
		cancels[n]()
	}()
	now := time.Now()
	<-mc.Done()
	assert.True(t, time.Since(now) >= 1*time.Second)
}

func TestValue(t *testing.T) {
	ctx1 := context.WithValue(context.Background(), "c1", "v1")
	ctx2 := context.WithValue(context.Background(), "c2", "v2")
	mc, _ := MergeContext(ctx1, ctx2)
	assert.Equal(t, mc.Value("c1").(string), "v1")
	assert.Equal(t, mc.Value("c2").(string), "v2")
	assert.Nil(t, mc.Value("c3"))
}

func TestErr(t *testing.T) {
	c1, ca1 := context.WithCancel(context.Background())
	c2, ca2 := context.WithCancel(context.Background())
	mc, _ := MergeContext(c1, c2)

	go func() {
		time.Sleep(1 * time.Second)
		cancels := []context.CancelFunc{ca1, ca2}
		n := rand.Int() % len(cancels)
		cancels[n]()
	}()
	assert.NoError(t, mc.Err())
	<-mc.Done()
	assert.Error(t, mc.Err())
}

func TestWithTimeout(t *testing.T) {
	mc, _ := MergeContext(context.Background(), context.Background())
	ctx, cancel := context.WithTimeout(mc, 1*time.Second)
	defer cancel()
	now := time.Now()
	<-ctx.Done()
	assert.True(t, time.Since(now) >= 1*time.Second)
}

func TestCancel(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	mc, cancel := MergeContext(context.Background(), ctx)
	now := time.Now()
	cancel()
	<-mc.Done()
	assert.True(t, time.Since(now) < 100*time.Millisecond)

	// call cancel() after done
	mc, cancel = MergeContext(context.Background(), ctx)
	time.Sleep(1010 * time.Millisecond)
	assert.NotPanics(t, assert.PanicTestFunc(cancel))
}
