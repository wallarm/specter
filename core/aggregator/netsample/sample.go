package netsample

import (
	"net"
	"net/url"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

const (
	ProtoCodeError          = 999
	DiscardedShootCodeError = 777
	DiscardedShootTag       = "discarded"
)

const (
	keyRTTMicro     = iota
	keyConnectMicro // TODO : set all for HTTP using httptrace and helper structs
	keySendMicro
	keyLatencyMicro
	keyReceiveMicro
	keyIntervalEventMicro // TODO: understand WTF is that mean and set it right.
	keyRequestBytes
	keyResponseBytes
	keyErrno
	keyProtoCode
	fieldsNum
)

func Acquire(tag string) *Sample {
	s := samplePool.Get().(*Sample)
	*s = Sample{
		timeStamp: time.Now(),
		tags:      tag,
	}
	return s
}

func releaseSample(s *Sample) { samplePool.Put(s) }

var samplePool = &sync.Pool{New: func() interface{} { return &Sample{} }}

type Sample struct {
	timeStamp time.Time
	tags      string
	id        uint64
	fields    [fieldsNum]int
	err       error
}

func (s *Sample) Tags() string { return s.tags }
func (s *Sample) AddTag(tag string) {
	if s.tags == "" {
		s.tags = tag
		return
	}
	s.tags += "|" + tag
}

func (s *Sample) ID() uint64      { return s.id }
func (s *Sample) SetID(id uint64) { s.id = id }

func (s *Sample) ProtoCode() int { return s.get(keyProtoCode) }
func (s *Sample) SetProtoCode(code int) {
	s.set(keyProtoCode, code)
	s.setRTT()
}

func (s *Sample) Err() error { return s.err }
func (s *Sample) SetErr(err error) {
	s.err = err
	s.set(keyErrno, getErrno(err))
	s.setRTT()
}

func (s *Sample) get(k int) int                      { return s.fields[k] }
func (s *Sample) set(k, v int)                       { s.fields[k] = v }
func (s *Sample) setDuration(k int, d time.Duration) { s.set(k, int(d.Nanoseconds()/1000)) }
func (s *Sample) setRTT() {
	if s.get(keyRTTMicro) == 0 {
		s.setDuration(keyRTTMicro, time.Since(s.timeStamp))
	}
}

func (s *Sample) SetUserDuration(d time.Duration) {
	s.setDuration(keyRTTMicro, d)
}

func (s *Sample) SetUserProto(code int) {
	s.set(keyProtoCode, code)
}

func (s *Sample) SetUserNet(code int) {
	s.set(keyErrno, code)
}

func (s *Sample) SetConnectTime(d time.Duration) {
	s.setDuration(keyConnectMicro, d)
}

func (s *Sample) SetSendTime(d time.Duration) {
	s.setDuration(keySendMicro, d)
}

func (s *Sample) SetLatency(d time.Duration) {
	s.setDuration(keyLatencyMicro, d)
}

func (s *Sample) SetReceiveTime(d time.Duration) {
	s.setDuration(keyReceiveMicro, d)
}

func (s *Sample) SetRequestBytes(b int) {
	s.set(keyRequestBytes, b)
}

func (s *Sample) SetResponseBytes(b int) {
	s.set(keyResponseBytes, b)
}

func (s *Sample) String() string {
	return string(appendPhout(s, nil, true))
}

func getErrno(err error) int {
	//
	if e, ok := err.(net.Error); ok && e.Timeout() {
		return 110 // Handle client Timeout as if it connection timeout
	}
	// stackerr.Error and etc.
	type hasUnderlying interface {
		Underlying() error
	}
	for {
		typed, ok := err.(hasUnderlying)
		if !ok {
			break
		}
		err = typed.Underlying()
	}
	err = errors.Cause(err)
	for {
		switch typed := err.(type) {
		case *net.OpError:
			err = typed.Err
		case *os.SyscallError:
			err = typed.Err
		case *url.Error:
			err = typed.Err
		case syscall.Errno:
			return int(typed)
		default:
			// Legacy default.
			return ProtoCodeError
		}
	}
}

func DiscardedShootSample() *Sample {
	sample := &Sample{
		timeStamp: time.Now(),
		tags:      DiscardedShootTag,
	}
	sample.SetUserNet(DiscardedShootCodeError)

	return sample
}
