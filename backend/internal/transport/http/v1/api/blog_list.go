package api

import (
	"github.com/gin-gonic/gin"
)

func (a *BlogAPI) GetUserBlogs(c *gin.Context) {
	a.blogDomainHandler.GetUserBlogs(c)
}

func (a *BlogAPI) BatchDeleteBlogs(c *gin.Context) {
	a.blogDomainHandler.BatchDeleteBlogs(c)
}
