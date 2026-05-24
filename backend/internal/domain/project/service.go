package project

import (
	"context"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"inkwords-backend/internal/infra/parser"
	"inkwords-backend/internal/prompt"
	"inkwords-backend/internal/service"
)

type Service struct {
	decomposition *service.DecompositionService
	gitFetcher    *parser.GitFetcher
	docParser     *parser.DocParser
	archiveParser *parser.ArchiveParser
	userService   *service.UserService
}

func NewService(decomposition *service.DecompositionService, gitFetcher *parser.GitFetcher, docParser *parser.DocParser, userService *service.UserService) *Service {
	return &Service{
		decomposition: decomposition,
		gitFetcher:    gitFetcher,
		docParser:     docParser,
		archiveParser: parser.NewArchiveParser(docParser),
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

	outlineRes, err := s.decomposition.GenerateOutline(ctx, content, prompt.ScenarioModeBeginnerWalkthrough, nil, nil)
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

func (s *Service) Parse(file io.Reader, filename string) (ParseResult, error) {
	// ZIP 解析会额外返回摘要信息，普通文件仍保持既有 source_content 语义不变。
	if strings.EqualFold(filepath.Ext(filename), ".zip") {
		result, err := s.archiveParser.ParseArchive(file, filename)
		if err != nil {
			return ParseResult{}, err
		}
		return ParseResult{
			SourceContent:  result.SourceContent,
			ArchiveSummary: result.ArchiveSummary,
		}, nil
	}

	content, err := s.docParser.Parse(file, filename)
	if err != nil {
		return ParseResult{}, err
	}
	return ParseResult{SourceContent: content}, nil
}
