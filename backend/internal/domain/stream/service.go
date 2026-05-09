package stream

import (
	"context"

	"github.com/google/uuid"

	"inkwords-backend/internal/service"
)

type Service struct {
	generator     *service.GeneratorService
	decomposition *service.DecompositionService
	userService   *service.UserService
}

func NewService(generator *service.GeneratorService, decomposition *service.DecompositionService, userService *service.UserService) *Service {
	return &Service{
		generator:     generator,
		decomposition: decomposition,
		userService:   userService,
	}
}

func (s *Service) CheckQuota(uid uuid.UUID) error {
	return s.userService.CheckQuota(uid)
}

func (s *Service) Generate(ctx context.Context, userID uuid.UUID, req GenerateRequest, chunkChan chan<- string, errChan chan<- error) {
	if len(req.Outline) > 0 {
		outline := make([]service.Chapter, 0, len(req.Outline))
		for _, ch := range req.Outline {
			outline = append(outline, service.Chapter{
				ID:      ch.ID,
				Title:   ch.Title,
				Summary: ch.Summary,
				Sort:    ch.Sort,
				Files:   ch.Files,
				Action:  ch.Action,
			})
		}

		var parentID uuid.UUID
		if req.ParentID != "" {
			if parsedID, err := uuid.Parse(req.ParentID); err == nil {
				parentID = parsedID
			}
		}
		if parentID == uuid.Nil {
			parentID = uuid.New()
		}
		s.decomposition.GenerateSeries(ctx, userID, parentID, req.SeriesTitle, outline, req.SourceContent, req.SourceType, req.GitURL, chunkChan, errChan)
		return
	}

	s.generator.GenerateBlogStream(ctx, userID, req.SourceContent, req.SourceType, chunkChan, errChan)
}

func (s *Service) Continue(bgCtx context.Context, userID uuid.UUID, blogID uuid.UUID, chunkChan chan<- string, errChan chan<- error) {
	s.decomposition.ContinueGeneration(bgCtx, userID, blogID, chunkChan, errChan)
}

func (s *Service) Polish(ctx context.Context, req PolishRequest, chunkChan chan<- string, errChan chan<- error) {
	s.generator.GeneratePolishDraftStream(ctx, req.Title, req.Content, chunkChan, errChan)
}

func (s *Service) AnalyzeStream(bgCtx context.Context, userID uuid.UUID, req GenerateRequest, progressChan chan<- string, errChan chan<- error) {
	if req.SourceType == "file" {
		s.decomposition.AnalyzeFileStream(bgCtx, userID, req.SourceContent, progressChan, errChan)
		return
	}
	s.decomposition.AnalyzeStream(bgCtx, userID, req.GitURL, req.SelectedModules, progressChan, errChan)
}

func (s *Service) ScanProjectModules(bgCtx context.Context, gitURL string, progressChan chan<- string) ([]ModuleCard, error) {
	modules, err := s.decomposition.ScanProjectModulesWithProgress(bgCtx, gitURL, progressChan)
	if err != nil {
		return nil, err
	}
	result := make([]ModuleCard, 0, len(modules))
	for _, m := range modules {
		result = append(result, ModuleCard{
			Path:        m.Path,
			Name:        m.Name,
			Description: m.Description,
		})
	}
	return result, nil
}
