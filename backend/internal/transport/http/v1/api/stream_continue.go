package api

import (
	"github.com/gin-gonic/gin"
)

// ContinueBlogStreamHandler proxies continuation SSE requests to the stream domain handler.
func (api *StreamAPI) ContinueBlogStreamHandler(c *gin.Context) {
	api.streamDomainHandler.ContinueBlogStreamHandler(c)
}
