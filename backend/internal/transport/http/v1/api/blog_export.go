package api

import (
	"github.com/gin-gonic/gin"
)

func (a *BlogAPI) ExportSeriesToObsidian(c *gin.Context) {
	a.blogDomainHandler.ExportSeriesToObsidian(c)
}

func (a *BlogAPI) ExportSeries(c *gin.Context) {
	a.blogDomainHandler.ExportSeries(c)
}

func (a *BlogAPI) ExportSeriesPDF(c *gin.Context) {
	a.blogDomainHandler.ExportSeriesPDF(c)
}

func (a *BlogAPI) ExportToObsidian(c *gin.Context) {
	a.blogDomainHandler.ExportToObsidian(c)
}
