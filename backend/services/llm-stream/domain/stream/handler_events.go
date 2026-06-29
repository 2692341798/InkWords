package stream

import (
	"io"
	"net/http"

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
