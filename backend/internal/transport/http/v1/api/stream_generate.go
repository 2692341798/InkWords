package api

import (
	"github.com/gin-gonic/gin"
)

// GenerateBlogStreamHandler proxies blog generation SSE requests to the stream domain handler.
func (api *StreamAPI) GenerateBlogStreamHandler(c *gin.Context) {
	api.streamDomainHandler.GenerateBlogStreamHandler(c)
}
