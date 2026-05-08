package api

import (
	"github.com/gin-gonic/gin"
)

func (api *StreamAPI) PolishBlogStreamHandler(c *gin.Context) {
	api.streamDomainHandler.PolishBlogStreamHandler(c)
}
