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
	ErrBlogNotFound               = errors.New("blog not found")
	ErrSeriesNotFound             = errors.New("series not found")
	ErrTextbookChapterNotFound    = errors.New("textbook chapter not found")
	ErrTextbookChapterNotApproved = errors.New("textbook chapter is not approved")
	ErrTextbookBookBuildNotFound  = errors.New("textbook book build not found")
	ErrTextbookBookBuildInvalid   = errors.New("textbook book build is invalid")
	ErrExportNotConfigured        = errors.New("export service is not configured")
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
	textbookRepo         TextbookChapterRepository
}

func NewService(repo Repository, storeFactory ObsidianStoreFactory, jsonGenerator JSONGenerator, model string, rootDir string, textbookRepos ...TextbookChapterRepository) *Service {
	if strings.TrimSpace(model) == "" {
		model = "deepseek-v4-flash"
	}
	if strings.TrimSpace(rootDir) == "" {
		rootDir = "wiki"
	}
	service := &Service{
		repo:                 repo,
		obsidianStoreFactory: storeFactory,
		jsonGenerator:        jsonGenerator,
		model:                strings.TrimSpace(model),
		rootDir:              strings.TrimSuffix(strings.TrimSpace(rootDir), "/"),
		now:                  time.Now,
	}
	if len(textbookRepos) > 0 {
		service.textbookRepo = textbookRepos[0]
	} else if textbookRepo, ok := repo.(TextbookChapterRepository); ok {
		service.textbookRepo = textbookRepo
	}
	return service
}

func (s *Service) GetSeriesBlogs(ctx context.Context, blogID uuid.UUID, workspaceID uuid.UUID) ([]Blog, error) {
	if s == nil || s.repo == nil {
		return nil, ErrExportNotConfigured
	}
	blogs, err := s.repo.GetSeriesBlogs(ctx, workspaceID, blogID)
	if err != nil {
		return nil, err
	}
	if len(blogs) == 0 {
		return nil, ErrSeriesNotFound
	}
	return blogs, nil
}

// GetApprovedTextbookChapter reads only the chapter revision explicitly approved
// by core-api. Draft and candidate revisions are not valid export inputs.
func (s *Service) GetApprovedTextbookChapter(ctx context.Context, workspaceID uuid.UUID, chapterID uuid.UUID) (TextbookChapterExport, error) {
	if s == nil || s.textbookRepo == nil {
		return TextbookChapterExport{}, ErrExportNotConfigured
	}
	return s.textbookRepo.GetApprovedTextbookChapter(ctx, workspaceID, chapterID)
}

// GetFrozenTextbookBookBuild returns the AST captured by core-api at build
// time. Renderers must use it instead of reading the current project state.
func (s *Service) GetFrozenTextbookBookBuild(ctx context.Context, workspaceID, buildID uuid.UUID) (TextbookBookBuildExport, error) {
	if s == nil || s.textbookRepo == nil {
		return TextbookBookBuildExport{}, ErrExportNotConfigured
	}
	return s.textbookRepo.GetTextbookBookBuild(ctx, workspaceID, buildID)
}

func (s *Service) getObsidianStore() (obsidian.Store, error) {
	if s == nil || s.obsidianStoreFactory == nil {
		return nil, ErrExportNotConfigured
	}
	return s.obsidianStoreFactory()
}
