package stream

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type testStreamWriter struct {
	*httptest.ResponseRecorder
	flushCount int
}

func newTestStreamWriter() *testStreamWriter {
	return &testStreamWriter{
		ResponseRecorder: httptest.NewRecorder(),
	}
}

func (w *testStreamWriter) CloseNotify() <-chan bool {
	ch := make(chan bool)
	return ch
}

func (w *testStreamWriter) Flush() {
	w.flushCount++
}

func (w *testStreamWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, nil
}

func (w *testStreamWriter) Pusher() http.Pusher {
	return nil
}

func (w *testStreamWriter) Status() int {
	return w.Code
}

func (w *testStreamWriter) Size() int {
	return w.Body.Len()
}

func (w *testStreamWriter) Written() bool {
	return w.Code != 0 || w.Body.Len() > 0
}

func (w *testStreamWriter) WriteHeaderNow() {}

func (w *testStreamWriter) WriteString(s string) (int, error) {
	return w.Body.WriteString(s)
}

func TestNewGenerateStreamChannels_UsesBufferedChannels(t *testing.T) {
	chunkChan, errChan := newGenerateStreamChannels()

	require.Greater(t, cap(chunkChan), 0)
	require.Greater(t, cap(errChan), 0)
}

func TestWriteStreamEvent_FlushesAfterWriting(t *testing.T) {
	gin.SetMode(gin.TestMode)

	writer := newTestStreamWriter()
	ctx, _ := gin.CreateTestContext(writer)

	writeStreamEvent(ctx, writer, "chunk", `{"status":"progress"}`)

	require.Equal(t, 1, writer.flushCount)
	require.Contains(t, writer.Body.String(), "event:chunk")
	require.Contains(t, writer.Body.String(), `data:{"status":"progress"}`)
}
