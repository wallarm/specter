// Code generated by mockery v2.20.2. DO NOT EDIT.

package coremock

import (
	io "io"

	mock "github.com/stretchr/testify/mock"
)

// DataSource is an autogenerated mock type for the DataSource type
type DataSource struct {
	mock.Mock
}

// OpenSource provides a mock function with given fields:
func (_m *DataSource) OpenSource() (io.ReadCloser, error) {
	ret := _m.Called()

	var r0 io.ReadCloser
	var r1 error
	if rf, ok := ret.Get(0).(func() (io.ReadCloser, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() io.ReadCloser); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(io.ReadCloser)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewDataSource interface {
	mock.TestingT
	Cleanup(func())
}

// NewDataSource creates a new instance of DataSource. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewDataSource(t mockConstructorTestingTNewDataSource) *DataSource {
	mock := &DataSource{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
