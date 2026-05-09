package api

import (
	"github.com/gin-gonic/gin"
)

func (api *StreamAPI) GenerateBlogStreamHandler(c *gin.Context) {
	api.streamDomainHandler.GenerateBlogStreamHandler(c)
}
