package datasink

import (
	"bytes"
	"io"

	"github.com/wallarm/specter/core"
	"github.com/wallarm/specter/lib/ioutil2"
)

type Buffer struct {
	bytes.Buffer
	ioutil2.NopCloser
}

var _ core.DataSink = &Buffer{}

func (b *Buffer) OpenSink() (wc io.WriteCloser, err error) {
	return b, nil
}

func NewBuffer() *Buffer {
	return &Buffer{}
}
