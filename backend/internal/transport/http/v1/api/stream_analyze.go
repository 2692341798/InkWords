package api

import (
	"github.com/gin-gonic/gin"
)

func (api *StreamAPI) AnalyzeStreamHandler(c *gin.Context) {
	api.streamDomainHandler.AnalyzeStreamHandler(c)
}
