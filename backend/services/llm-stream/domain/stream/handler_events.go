package stream

import (
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func newGenerateStreamChannels() (chan string, chan error) {
	return make(chan string, streamChannelBufferSize), make(chan error, 1)
}

func writeStreamEvent(c *gin.Context, w io.Writer, event string, payload interface{}) {
	c.SSEvent(event, payload)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// drainStreamChannels safely drains chunk and error channels on context cancellation
// to prevent goroutine leaks in SSE stream handlers.
func drainStreamChannels(chunkChan chan string, errChan chan error) {
	chunkOpen, errOpen := true, true
	for chunkOpen || errOpen {
		select {
		case _, ok := <-chunkChan:
			if !ok {
				chunkOpen = false
			}
		case _, ok := <-errChan:
			if !ok {
				errOpen = false
			}
		}
	}
}

// sseStreamBody runs the shared SSE event loop used by generate/continue/polish/analyze handlers.
func sseStreamBody(c *gin.Context, chunkChan chan string, errChan *chan error, operation streamOperation) {
	ctx := c.Request.Context()

	c.Stream(func(w io.Writer) bool {
		select {
		case <-ctx.Done():
			go drainStreamChannels(chunkChan, *errChan)
			return false
		case err, ok := <-*errChan:
			if ok && err != nil {
				writeStreamEvent(c, w, "error", externalStreamErrorMessage(operation, err))
				return false
			}
			if !ok {
				*errChan = nil
			}
			return true
		case chunk, ok := <-chunkChan:
			if !ok {
				writeStreamEvent(c, w, "done", "[DONE]")
				return false
			}
			writeStreamEvent(c, w, "chunk", chunk)
			return true
		case <-time.After(10 * time.Second):
			writeStreamEvent(c, w, "ping", "keepalive")
			return true
		}
	})
}
