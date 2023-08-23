package coretest

import (
	"io"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wallarm/specter/core"
)

func AssertSinkEqualStdStream(t *testing.T, expectedPtr **os.File, getSink func() core.DataSink) {
	temp, err := os.CreateTemp("", "")
	require.NoError(t, err)

	backup := *expectedPtr
	defer func() {
		*expectedPtr = backup
	}()
	*expectedPtr = temp
	const testdata = "wall"

	wc, err := getSink().OpenSink()
	require.NoError(t, err)

	_, err = io.WriteString(wc, testdata)
	require.NoError(t, err)

	err = wc.Close()
	require.NoError(t, err)

	_, _ = temp.Seek(0, io.SeekStart)
	data, _ := io.ReadAll(temp)
	assert.Equal(t, testdata, string(data))
}

func AssertSinkEqualFile(t *testing.T, fs afero.Fs, filename string, sink core.DataSink) {
	_ = afero.WriteFile(fs, filename, []byte("should be truncated"), 0644)

	wc, err := sink.OpenSink()
	require.NoError(t, err)

	const testdata = "abcd"

	_, err = io.WriteString(wc, testdata)
	require.NoError(t, err)

	err = wc.Close()
	require.NoError(t, err)

	data, err := afero.ReadFile(fs, filename)
	require.NoError(t, err)

	assert.Equal(t, testdata, string(data))
}
