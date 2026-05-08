package project

import (
	"context"
	"io"
	"strings"

	"github.com/google/uuid"

	"inkwords-backend/internal/parser"
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

func (s *Service) ScanProjectModules(ctx context.Context, gitURL string) (interface{}, error) {
	return s.decomposition.ScanProjectModules(ctx, gitURL)
}

func (s *Service) Analyze(ctx context.Context, gitURL string, subDir string) (outline interface{}, sourceContent string, stage string, err error) {
	treeContent, chunks, err := s.gitFetcher.FetchWithSubDir(gitURL, subDir, nil)
	if err != nil {
		return nil, "", "fetch", err
	}

	var fullContentBuilder strings.Builder
	fullContentBuilder.WriteString(treeContent)
	fullContentBuilder.WriteString("\n=== Repository Content ===\n")
	for _, chunk := range chunks {
		fullContentBuilder.WriteString(chunk.Content)
	}
	content := fullContentBuilder.String()

	outlineRes, err := s.decomposition.GenerateOutline(ctx, content, nil, nil)
	if err != nil {
		return nil, content, "outline", err
	}

	return outlineRes, content, "", nil
}

func (s *Service) Parse(file io.Reader, filename string) (string, error) {
	return s.docParser.Parse(file, filename)
}
