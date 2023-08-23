package schedule

import (
	"sync"
	"time"

	"github.com/wallarm/specter/core"
)

type CompositeConf struct {
	Nested []core.Schedule `config:"nested"`
}

func NewCompositeConf(conf CompositeConf) core.Schedule {
	return NewComposite(conf.Nested...)
}

func NewComposite(scheds ...core.Schedule) core.Schedule {
	switch len(scheds) {
	case 0:
		return NewOnce(0)
	case 1:
		return scheds[0]
	}

	var (
		left            = make([]int, len(scheds))
		unknown         bool // If meet any Left() < 0, all previous leftBefore is unknown
		leftAccumulator int  // If unknown, then at least leftBefore accumulated, else exactly leftBefore.
	)
	for i := len(scheds) - 1; i >= 0; i-- {
		left[i] = leftAccumulator
		schedLeft := scheds[i].Left()
		if schedLeft < 0 {
			schedLeft = -1
			unknown = true
			leftAccumulator = -1
		}
		if !unknown {
			leftAccumulator += schedLeft
		}
	}

	return &compositeSchedule{
		scheds:    scheds,
		leftAfter: left,
	}
}

type compositeSchedule struct {
	// Under read lock, goroutine can read slices, it's values, and call values goroutine safe methods.
	// Under write lock, goroutine can do anything.
	rwMu      sync.RWMutex
	scheds    []core.Schedule // At least once schedule. First schedule can be finished.
	leftAfter []int           // Tokens leftBefore, if known exactly, or at least tokens leftBefore otherwise.
}

func (s *compositeSchedule) Start(startAt time.Time) {
	s.rwMu.Lock()
	defer s.rwMu.Unlock()
	s.scheds[0].Start(startAt)
}
func (s *compositeSchedule) Next() (tx time.Time, ok bool) {
	s.rwMu.RLock()
	tx, ok = s.scheds[0].Next()
	if ok {
		s.rwMu.RUnlock()
		return // Got token, all is good.
	}
	schedsLeft := len(s.scheds)
	s.rwMu.RUnlock()
	if schedsLeft == 1 {
		return // All nested schedules has been finished, so composite is finished too.
	}
	// Current schedule is finished, but some are left.
	// Let's start next, with got finish time from previous!
	s.rwMu.Lock()
	schedsLeftNow := len(s.scheds)
	somebodyStartedNextBeforeUs := schedsLeftNow < schedsLeft
	if somebodyStartedNextBeforeUs {
		// Let's just take token.
		tx, ok = s.scheds[0].Next()
		s.rwMu.Unlock()
		if ok || schedsLeftNow == 1 {
			return
		}
		// Very strange. Schedule was started and drained while we was waiting for it.
		// Should very rare, so let's just retry.
		return s.Next()
	}
	s.startNext(tx)
	tx, ok = s.scheds[0].Next()
	s.rwMu.Unlock()
	if !ok && schedsLeftNow > 1 {
		// What? Schedule without any tokens? Okay, just retry.
		return s.Next()
	}
	return
}

func (s *compositeSchedule) Left() int {
	s.rwMu.RLock()
	schedsLeft := len(s.scheds)
	leftAfter := int(s.leftAfter[0])
	left := s.scheds[0].Left()
	s.rwMu.RUnlock()
	if schedsLeft == 1 {
		return left
	}
	if left == 0 {
		if leftAfter >= 0 {
			return leftAfter
		}
		// leftAfter was unknown, at schedule create moment.
		// But now, it can be finished. Let's shift, and try one more time.
		s.rwMu.Lock()
		shedsLeftNow := len(s.scheds)
		if shedsLeftNow == schedsLeft {
			currentFinishTime, ok := s.scheds[0].Next()
			if ok {
				s.rwMu.Unlock()
				panic("current schedule is not finished")
			}
			s.startNext(currentFinishTime)
		}
		s.rwMu.Unlock()
		return s.Left()
	}
	if left < 0 {
		return -1
	}
	return left + leftAfter
}

func (s *compositeSchedule) startNext(currentFinishTime time.Time) {
	s.scheds = s.scheds[1:]
	s.leftAfter = s.leftAfter[1:]
	s.scheds[0].Start(currentFinishTime)
}
