package project

import (
	"context"
	"io"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"inkwords-backend/shared/kernel/prompt"
	"inkwords-backend/shared/platform/parser"
)

type Service struct {
	analyzer      projectAnalyzer
	gitFetcher    *parser.GitFetcher
	docParser     *parser.DocParser
	archiveParser *parser.ArchiveParser
	quotaChecker  quotaChecker
}

type projectAnalyzer interface {
	ScanProjectModules(ctx context.Context, gitURL string) ([]ModuleCard, error)
	GenerateOutline(ctx context.Context, sourceContent string, scenarioMode prompt.ScenarioMode) (OutlineResult, error)
}

type quotaChecker interface {
	CheckQuota(uid uuid.UUID) error
}

func NewService(analyzer projectAnalyzer, gitFetcher *parser.GitFetcher, docParser *parser.DocParser, quotaChecker quotaChecker) *Service {
	return &Service{
		analyzer:      analyzer,
		gitFetcher:    gitFetcher,
		docParser:     docParser,
		archiveParser: parser.NewArchiveParser(docParser),
		quotaChecker:  quotaChecker,
	}
}

func (s *Service) CheckQuota(uid uuid.UUID) error {
	if s.quotaChecker == nil {
		return nil
	}
	return s.quotaChecker.CheckQuota(uid)
}

func (s *Service) ScanProjectModules(ctx context.Context, gitURL string) ([]ModuleCard, error) {
	return s.analyzer.ScanProjectModules(ctx, gitURL)
}

func (s *Service) Analyze(ctx context.Context, gitURL string, subDir string) (outline OutlineResult, sourceContent string, stage string, err error) {
	treeContent, chunks, err := s.gitFetcher.FetchWithSubDir(gitURL, subDir, nil)
	if err != nil {
		return OutlineResult{}, "", "fetch", err
	}
	content := AssembleSourceContent(treeContent, chunks)

	outlineRes, err := s.analyzer.GenerateOutline(ctx, content, prompt.ScenarioModeBeginnerWalkthrough)
	if err != nil {
		return OutlineResult{}, content, "outline", err
	}
	return outlineRes, content, "", nil
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
