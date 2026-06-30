package task

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("client disconnected")
}

func TestCopyDownload_ReturnsWriterFailure(t *testing.T) {
	err := copyDownload(failingWriter{}, strings.NewReader("pdf"))
	require.ErrorContains(t, err, "stream download")
	require.ErrorContains(t, err, "client disconnected")
}
