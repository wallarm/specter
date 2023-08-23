package ginkgoutil

import (
	"reflect"

	"github.com/onsi/gomega"
)

func ExpectFuncsEqual(f1, f2 interface{}) {
	val1 := reflect.ValueOf(f1)
	val2 := reflect.ValueOf(f2)
	gomega.Expect(val1.Pointer()).To(gomega.Equal(val2.Pointer()))
}
