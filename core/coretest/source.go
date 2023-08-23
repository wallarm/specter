package coretest

import (
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wallarm/specter/core"
	"github.com/wallarm/specter/lib/testutil"
)

func AssertSourceEqualStdStream(t *testing.T, expectedPtr **os.File, getSource func() core.DataSource) {
	temp, err := ioutil.TempFile("", "")
	require.NoError(t, err)

	backup := *expectedPtr
	defer func() {
		*expectedPtr = backup
	}()
	*expectedPtr = temp
	const testdata = "abcd"
	_, err = io.WriteString(temp, testdata)
	require.NoError(t, err)

	rc, err := getSource().OpenSource()
	require.NoError(t, err)

	err = rc.Close()
	require.NoError(t, err, "std stream should not be closed")

	_, _ = temp.Seek(0, io.SeekStart)
	data, _ := io.ReadAll(temp)
	assert.Equal(t, testdata, string(data))
}

func AssertSourceEqualFile(t *testing.T, fs afero.Fs, filename string, source core.DataSource) {
	const testdata = "abcd"
	_ = afero.WriteFile(fs, filename, []byte(testdata), 0644)

	rc, err := source.OpenSource()
	require.NoError(t, err)

	data := testutil.ReadString(t, rc)
	err = rc.Close()
	require.NoError(t, err)

	assert.Equal(t, testdata, data)
}
