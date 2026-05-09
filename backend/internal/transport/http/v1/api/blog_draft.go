package api

import (
	"github.com/gin-gonic/gin"
)

func (a *BlogAPI) CreateDraftBlog(c *gin.Context) {
	a.blogDomainHandler.CreateDraftBlog(c)
}
