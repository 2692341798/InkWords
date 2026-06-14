package export

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	platformllm "inkwords-backend/shared/platform/llm"
	"inkwords-backend/shared/platform/obsidian"
)

var (
	ErrBlogNotFound        = errors.New("blog not found")
	ErrSeriesNotFound      = errors.New("series not found")
	ErrExportNotConfigured = errors.New("export service is not configured")
)

type ObsidianStoreFactory func() (obsidian.Store, error)

type JSONGenerator interface {
	GenerateJSON(ctx context.Context, model string, messages []platformllm.Message) (string, error)
}

// Service owns export-service's PDF and Obsidian export workflows.
type Service struct {
	repo                 Repository
	obsidianStoreFactory ObsidianStoreFactory
	jsonGenerator        JSONGenerator
	model                string
	rootDir              string
	now                  func() time.Time
}

func NewService(repo Repository, storeFactory ObsidianStoreFactory, jsonGenerator JSONGenerator, model string, rootDir string) *Service {
	if strings.TrimSpace(model) == "" {
		model = "deepseek-v4-flash"
	}
	if strings.TrimSpace(rootDir) == "" {
		rootDir = "wiki"
	}
	return &Service{
		repo:                 repo,
		obsidianStoreFactory: storeFactory,
		jsonGenerator:        jsonGenerator,
		model:                strings.TrimSpace(model),
		rootDir:              strings.TrimSuffix(strings.TrimSpace(rootDir), "/"),
		now:                  time.Now,
	}
}

func (s *Service) GetSeriesBlogs(ctx context.Context, blogID uuid.UUID, userID uuid.UUID) ([]Blog, error) {
	if s == nil || s.repo == nil {
		return nil, ErrExportNotConfigured
	}
	blogs, err := s.repo.GetSeriesBlogs(ctx, userID, blogID)
	if err != nil {
		return nil, err
	}
	if len(blogs) == 0 {
		return nil, ErrSeriesNotFound
	}
	return blogs, nil
}

func (s *Service) getObsidianStore() (obsidian.Store, error) {
	if s == nil || s.obsidianStoreFactory == nil {
		return nil, ErrExportNotConfigured
	}
	return s.obsidianStoreFactory()
}
