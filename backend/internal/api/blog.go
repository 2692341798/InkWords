package api

import (
"net/http"
"strconv"

"github.com/gin-gonic/gin"
"github.com/google/uuid"

"inkwords-backend/internal/service"
)

type BlogAPI struct {
blogService *service.BlogService
}

func NewBlogAPI() *BlogAPI {
return &BlogAPI{
blogService: service.NewBlogService(),
}
}

// GetUserBlogs 获取当前用户的博客列表
func (a *BlogAPI) GetUserBlogs(c *gin.Context) {
userIDStr, exists := c.Get("user_id")
if !exists {
c.JSON(http.StatusUnauthorized, gin.H{
"code":    http.StatusUnauthorized,
"message": "unauthorized",
"data":    nil,
})
return
}

uid, ok := userIDStr.(uuid.UUID)
if !ok {
c.JSON(http.StatusInternalServerError, gin.H{
"code":    http.StatusInternalServerError,
"message": "invalid user id type",
"data":    nil,
})
return
}

page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
if page < 1 {
page = 1
}
if size < 1 || size > 100 {
size = 20
}

blogs, err := a.blogService.GetUserBlogs(c.Request.Context(), uid, page, size)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{
"code":    http.StatusInternalServerError,
"message": err.Error(),
"data":    nil,
})
return
}

c.JSON(http.StatusOK, gin.H{
"code":    http.StatusOK,
"message": "success",
"data":    blogs,
})
}

// UpdateBlog 更新博客内容
func (a *BlogAPI) UpdateBlog(c *gin.Context) {
userIDStr, exists := c.Get("user_id")
if !exists {
c.JSON(http.StatusUnauthorized, gin.H{
"code":    http.StatusUnauthorized,
"message": "unauthorized",
"data":    nil,
})
return
}

uid, ok := userIDStr.(uuid.UUID)
if !ok {
c.JSON(http.StatusInternalServerError, gin.H{
"code":    http.StatusInternalServerError,
"message": "invalid user id type",
"data":    nil,
})
return
}

blogIDStr := c.Param("id")
blogID, err := uuid.Parse(blogIDStr)
if err != nil {
c.JSON(http.StatusBadRequest, gin.H{
"code":    http.StatusBadRequest,
"message": "invalid blog id",
"data":    nil,
})
return
}

var req service.UpdateBlogRequest
if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{
"code":    http.StatusBadRequest,
"message": "invalid request body",
"data":    nil,
})
return
}

if err := a.blogService.UpdateBlog(c.Request.Context(), blogID, uid, req); err != nil {
c.JSON(http.StatusInternalServerError, gin.H{
"code":    http.StatusInternalServerError,
"message": err.Error(),
"data":    nil,
})
return
}

c.JSON(http.StatusOK, gin.H{
"code":    http.StatusOK,
"message": "success",
"data":    nil,
})
}
