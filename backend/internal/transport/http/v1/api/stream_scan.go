package api

import (
	"github.com/gin-gonic/gin"
)

func (api *StreamAPI) ScanStreamHandler(c *gin.Context) {
	api.streamDomainHandler.ScanStreamHandler(c)
}
