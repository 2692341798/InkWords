package api

import (
	"github.com/gin-gonic/gin"
)

// PolishBlogStreamHandler proxies polish SSE requests to the stream domain handler.
func (api *StreamAPI) PolishBlogStreamHandler(c *gin.Context) {
	api.streamDomainHandler.PolishBlogStreamHandler(c)
}
