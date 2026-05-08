package api

import (
	"github.com/gin-gonic/gin"
)

func (api *StreamAPI) ContinueBlogStreamHandler(c *gin.Context) {
	api.streamDomainHandler.ContinueBlogStreamHandler(c)
}
