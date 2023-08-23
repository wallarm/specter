package coreutil

import (
	"context"
	"time"

	"github.com/wallarm/specter/core"
)

// Waiter goroutine unsafe wrapper for efficient waiting schedule.
type Waiter struct {
	sched         core.Schedule
	ctx           context.Context
	slowDownItems int

	// Lazy initialized.
	timer   *time.Timer
	lastNow time.Time
}

func NewWaiter(sched core.Schedule, ctx context.Context) *Waiter {
	return &Waiter{sched: sched, ctx: ctx}
}

// Wait waits for next waiter schedule event.
// Returns true, if event successfully waited, or false
// if waiter context is done, or schedule finished.
func (w *Waiter) Wait() (ok bool) {
	// Check, that context is not done. Very quick: 5 ns for op, due to benchmark.
	select {
	case <-w.ctx.Done():
		w.slowDownItems = 0
		return false
	default:
	}
	next, ok := w.sched.Next()
	if !ok {
		w.slowDownItems = 0
		return false
	}
	// Get current time lazily.
	// For once schedule, for example, we need to get it only once.
	if next.Before(w.lastNow) {
		w.slowDownItems++
		return true
	}
	w.lastNow = time.Now()
	waitFor := next.Sub(w.lastNow)
	if waitFor <= 0 {
		w.slowDownItems++
		return true
	}
	w.slowDownItems = 0
	// Lazy init. We don't need timer for unlimited and once schedule.
	if w.timer == nil {
		w.timer = time.NewTimer(waitFor)
	} else {
		w.timer.Reset(waitFor)
	}
	select {
	case <-w.timer.C:
		return true
	case <-w.ctx.Done():
		return false
	}
}

// IsSlowDown returns true, if schedule contains 2 elements before current time.
func (w *Waiter) IsSlowDown() (ok bool) {
	select {
	case <-w.ctx.Done():
		return false
	default:
		return w.slowDownItems >= 2
	}
}

// IsFinished is quick check, that wait context is not canceled and there are some tokens left in
// schedule.
func (w *Waiter) IsFinished() (ok bool) {
	select {
	case <-w.ctx.Done():
		return true
	default:
		return w.sched.Left() == 0
	}
}
