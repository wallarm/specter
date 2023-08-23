package schedule

import (
	"time"

	"github.com/wallarm/specter/core"
)

// NewUnlimited returns schedule that generates unlimited ops for passed duration.
func NewUnlimited(duration time.Duration) core.Schedule {
	return &unlimitedSchedule{duration: duration}
}

type UnlimitedConfig struct {
	Duration time.Duration `validate:"min-time=1ms"`
}

func NewUnlimitedConf(conf UnlimitedConfig) core.Schedule {
	return NewUnlimited(conf.Duration)
}

type unlimitedSchedule struct {
	duration time.Duration

	StartSync
	finish time.Time
}

func (s *unlimitedSchedule) Start(startAt time.Time) {
	s.MarkStarted()
	s.startOnce.Do(func() {
		s.finish = startAt.Add(s.duration)
	})
}

func (s *unlimitedSchedule) Next() (tx time.Time, ok bool) {
	s.startOnce.Do(func() {
		s.MarkStarted()
		s.finish = time.Now().Add(s.duration)
	})
	now := time.Now()
	if now.Before(s.finish) {
		return now, true
	}
	return s.finish, false
}

func (s *unlimitedSchedule) Left() int {
	if !s.IsStarted() || time.Now().Before(s.finish) {
		return -1
	}
	return 0
}
