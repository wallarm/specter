package testutil

import (
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
)

func ReadString(t TestingT, r io.Reader) string {
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(data)
}

func ReadFileString(t TestingT, fs afero.Fs, name string) string {
	getHelper(t).Helper()
	data, err := afero.ReadFile(fs, name)
	require.NoError(t, err)
	return string(data)

}

func AssertFileEqual(t TestingT, fs afero.Fs, name string, expected string) {
	getHelper(t).Helper()
	actual := ReadFileString(t, fs, name)
	assert.Equal(t, expected, actual)
}
