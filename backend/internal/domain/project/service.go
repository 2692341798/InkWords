package project

import (
	"context"
	"io"

	"github.com/google/uuid"

	"inkwords-backend/internal/infra/parser"
	"inkwords-backend/internal/service"
)

type Service struct {
	decomposition *service.DecompositionService
	gitFetcher    *parser.GitFetcher
	docParser     *parser.DocParser
	userService   *service.UserService
}

func NewService(decomposition *service.DecompositionService, gitFetcher *parser.GitFetcher, docParser *parser.DocParser, userService *service.UserService) *Service {
	return &Service{
		decomposition: decomposition,
		gitFetcher:    gitFetcher,
		docParser:     docParser,
		userService:   userService,
	}
}

func (s *Service) CheckQuota(uid uuid.UUID) error {
	return s.userService.CheckQuota(uid)
}

func (s *Service) ScanProjectModules(ctx context.Context, gitURL string) ([]ModuleCard, error) {
	modules, err := s.decomposition.ScanProjectModules(ctx, gitURL)
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

func (s *Service) Analyze(ctx context.Context, gitURL string, subDir string) (outline OutlineResult, sourceContent string, stage string, err error) {
	treeContent, chunks, err := s.gitFetcher.FetchWithSubDir(gitURL, subDir, nil)
	if err != nil {
		return OutlineResult{}, "", "fetch", err
	}
	content := AssembleSourceContent(treeContent, chunks)

	outlineRes, err := s.decomposition.GenerateOutline(ctx, content, nil, nil)
	if err != nil {
		return OutlineResult{}, content, "outline", err
	}

	chapters := make([]Chapter, 0, len(outlineRes.Chapters))
	for _, ch := range outlineRes.Chapters {
		chapters = append(chapters, Chapter{
			ID:      ch.ID,
			Title:   ch.Title,
			Summary: ch.Summary,
			Sort:    ch.Sort,
			Files:   ch.Files,
			Action:  ch.Action,
		})
	}

	return OutlineResult{
		SeriesTitle: outlineRes.SeriesTitle,
		Chapters:    chapters,
		ParentID:    outlineRes.ParentID,
	}, content, "", nil
}

func (s *Service) Parse(file io.Reader, filename string) (string, error) {
	return s.docParser.Parse(file, filename)
}
