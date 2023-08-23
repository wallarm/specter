package coreutil

import "github.com/wallarm/specter/core"

func ReturnSampleIfBorrowed(s core.Sample) {
	borrowed, ok := s.(core.BorrowedSample)
	if !ok {
		return
	}
	borrowed.Return()
}
