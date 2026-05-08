package api

import (
	"github.com/gin-gonic/gin"
)

func (a *BlogAPI) UpdateBlog(c *gin.Context) {
	a.blogDomainHandler.UpdateBlog(c)
}
