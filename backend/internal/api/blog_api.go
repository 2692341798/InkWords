package api

import "inkwords-backend/internal/service"

type BlogAPI struct {
	blogService *service.BlogService
}

func NewBlogAPI() *BlogAPI {
	return &BlogAPI{
		blogService: service.NewBlogService(),
	}
}

