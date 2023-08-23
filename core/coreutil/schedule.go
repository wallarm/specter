package coreutil

import (
	"sync"
	"time"

	"github.com/wallarm/specter/core"
)

// NewCallbackOnFinishSchedule returns schedule that calls back once onFinish
// just before first callee could know, that schedule is finished.
// That is, calls onFinish once, first time, whet Next() returns ok == false
// or Left() returns 0.
func NewCallbackOnFinishSchedule(s core.Schedule, onFinish func()) core.Schedule {
	return &callbackOnFinishSchedule{
		Schedule: s,
		onFinish: onFinish,
	}
}

type callbackOnFinishSchedule struct {
	core.Schedule
	onFinishOnce sync.Once
	onFinish     func()
}

func (s *callbackOnFinishSchedule) Next() (ts time.Time, ok bool) {
	ts, ok = s.Schedule.Next()
	if !ok {
		s.onFinishOnce.Do(s.onFinish)
	}
	return
}

func (s *callbackOnFinishSchedule) Left() int {
	left := s.Schedule.Left()
	if left == 0 {
		s.onFinishOnce.Do(s.onFinish)
	}
	return left
}
