package aggregator

import (
	"bufio"
	"io"

	jsoniter "github.com/json-iterator/go"
	"github.com/wallarm/specter/core"
	"github.com/wallarm/specter/core/coreutil"
	"github.com/wallarm/specter/lib/ioutil2"
)

type JSONLineAggregatorConfig struct {
	EncoderAggregatorConfig `config:",squash"`
	JSONLineEncoderConfig   `config:",squash"`
}

type JSONLineEncoderConfig struct {
	JSONIterConfig            `config:",squash"`
	coreutil.BufferSizeConfig `config:",squash"`
}

// JSONIterConfig is subset of jsoniter.Config that may be useful to configure.
type JSONIterConfig struct {
	// MarshalFloatWith6Digits makes float marshalling faster.
	MarshalFloatWith6Digits bool `config:"marshal-float-with-6-digits"`
	// SortMapKeys useful, when sample contains map object, and you want to see them in same order.
	SortMapKeys bool `config:"sort-map-keys"`
}

func DefaultJSONLinesAggregatorConfig() JSONLineAggregatorConfig {
	return JSONLineAggregatorConfig{
		EncoderAggregatorConfig: DefaultEncoderAggregatorConfig(),
	}
}

// Aggregates samples in JSON Lines format: each output line is a Valid JSON Value of one sample.
// See http://jsonlines.org/ for details.
func NewJSONLinesAggregator(conf JSONLineAggregatorConfig) core.Aggregator {
	var newEncoder NewSampleEncoder = func(w io.Writer, onFlush func()) SampleEncoder {
		w = ioutil2.NewCallbackWriter(w, onFlush)
		return NewJSONEncoder(w, conf.JSONLineEncoderConfig)
	}
	return NewEncoderAggregator(newEncoder, conf.EncoderAggregatorConfig)
}

func NewJSONEncoder(w io.Writer, conf JSONLineEncoderConfig) SampleEncoder {
	apiConfig := jsoniter.Config{
		SortMapKeys:             conf.JSONIterConfig.SortMapKeys,
		MarshalFloatWith6Digits: conf.JSONIterConfig.MarshalFloatWith6Digits,
	}

	api := apiConfig.Froze()
	// NOTE: internal buffering is not working really. Don't know why
	// OPTIMIZE: don't wrap into buffer, if already ioutil2.ByteWriter
	buf := bufio.NewWriterSize(w, conf.BufferSizeOrDefault())
	stream := jsoniter.NewStream(api, buf, conf.BufferSizeOrDefault())
	return &jsonEncoder{stream, buf}
}

type jsonEncoder struct {
	*jsoniter.Stream
	buf *bufio.Writer
}

func (e *jsonEncoder) Encode(s core.Sample) error {
	e.WriteVal(s)
	e.WriteRaw("\n")
	return e.Error
}

func (e *jsonEncoder) Flush() error {
	err := e.Stream.Flush()
	_ = e.buf.Flush()
	return err
}
