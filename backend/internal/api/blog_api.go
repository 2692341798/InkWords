package api

import (
	"inkwords-backend/internal/db"
	blogdomain "inkwords-backend/internal/domain/blog"
	"inkwords-backend/internal/service"
)

type BlogAPI struct {
	blogService       *service.BlogService
	blogDomainHandler *blogdomain.Handler
}

func NewBlogAPI() *BlogAPI {
	repo := blogdomain.NewGormRepository(db.DB)
	domainService := blogdomain.NewService(repo)
	blogService := service.NewBlogService()

	return &BlogAPI{
		blogService:       blogService,
		blogDomainHandler: blogdomain.NewHandlerWithLegacy(domainService, blogService),
	}
}

// NewBlogAPIWithDeps 创建 BlogAPI，并允许从外部注入依赖（用于 cmd/server 统一组装 DI）。
func NewBlogAPIWithDeps(blogService *service.BlogService, blogDomainHandler *blogdomain.Handler) *BlogAPI {
	return &BlogAPI{
		blogService:       blogService,
		blogDomainHandler: blogDomainHandler,
	}
}
