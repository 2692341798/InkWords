package stream

import (
	"context"
	"errors"

	"github.com/google/uuid"

	sharedblog "inkwords-backend/shared/kernel/blog"
	"inkwords-backend/shared/kernel/prompt"
)

// GenerateSingleResult 表示单篇生成完成后交给调用方的结构化结果。
type GenerateSingleResult struct {
	ResultJSON []byte
}

// ContinueTaskResultSnapshot 表示续写任务完成后交给调用方的业务事实快照。
type ContinueTaskResultSnapshot struct {
	BlogID          string
	AppendedContent string
	FinalContent    string
	EstimatedTokens int
	Usage           TaskResultUsage
}

// Generator 定义单篇博客生成与润色的能力边界。
type Generator interface {
	GenerateBlogStreamWithProfile(ctx context.Context, userID uuid.UUID, sourceContent string, sourceType string, scenarioMode prompt.ScenarioMode, style string, profile prompt.PromptProfile, chunkChan chan<- string, errChan chan<- error)
	GeneratePolishDraftStream(ctx context.Context, title string, content string, chunkChan chan<- string, errChan chan<- error)
	BuildGenerateSingleTaskResult(ctx context.Context, sourceType string, content string) (GenerateSingleResult, error)
}

// Decomposition 定义系列生成、续写、分析与扫描的能力边界。
type Decomposition interface {
	GenerateSeriesWithProfile(ctx context.Context, userID uuid.UUID, parentID uuid.UUID, seriesTitle string, outline []sharedblog.Chapter, sourceContent string, sourceType string, gitURL string, scenarioMode prompt.ScenarioMode, style string, profile prompt.PromptProfile, chunkChan chan<- string, errChan chan<- error)
	ContinueGeneration(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, chunkChan chan<- string, errChan chan<- error)
	BuildContinueTaskResult(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, appendedContent string) (ContinueTaskResultSnapshot, error)
	TakeGenerateSeriesTaskResult(parentID uuid.UUID) ([]byte, error)
	AnalyzeStream(ctx context.Context, userID uuid.UUID, gitURL string, selectedModules []string, scenarioMode prompt.ScenarioMode, progressChan chan<- string, errChan chan<- error)
	AnalyzeFileStream(ctx context.Context, userID uuid.UUID, sourceContent string, scenarioMode prompt.ScenarioMode, progressChan chan<- string, errChan chan<- error)
	ScanProjectModulesWithProgress(ctx context.Context, gitURL string, progressChan chan<- string) ([]ModuleCard, error)
}

// QuotaChecker 定义用量配额检查的能力边界。
type QuotaChecker interface {
	CheckQuota(uid uuid.UUID) error
}

type Service struct {
	generator    Generator
	decomposition Decomposition
	quotaChecker QuotaChecker
}

func NewService(generator Generator, decomposition Decomposition, quotaChecker QuotaChecker) *Service {
	return &Service{
		generator:    generator,
		decomposition: decomposition,
		quotaChecker: quotaChecker,
	}
}

func (s *Service) CheckQuota(uid uuid.UUID) error {
	return s.quotaChecker.CheckQuota(uid)
}

func (s *Service) Generate(ctx context.Context, userID uuid.UUID, req GenerateRequest, chunkChan chan<- string, errChan chan<- error) {
	style := req.ArticleStyle
	if style == "" {
		style = "general"
	}
	scenarioMode := prompt.ScenarioMode(req.ScenarioMode)
	profile := prompt.ResolvePromptProfileKey(req.PromptProfileKey, scenarioMode)

	if len(req.Outline) > 0 {
		outline := make([]sharedblog.Chapter, 0, len(req.Outline))
		for _, ch := range req.Outline {
			outline = append(outline, sharedblog.Chapter{
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
		s.decomposition.GenerateSeriesWithProfile(
			ctx,
			userID,
			parentID,
			req.SeriesTitle,
			outline,
			req.SourceContent,
			req.SourceType,
			req.GitURL,
			scenarioMode,
			style,
			profile,
			chunkChan,
			errChan,
		)
		return
	}

	s.generator.GenerateBlogStreamWithProfile(
		ctx,
		userID,
		req.SourceContent,
		req.SourceType,
		scenarioMode,
		style,
		profile,
		chunkChan,
		errChan,
	)
}

// BuildGenerateSingleTaskResult 基于单篇生成最终正文构造结构化任务结果。
func (s *Service) BuildGenerateSingleTaskResult(ctx context.Context, req GenerateRequest, content string) ([]byte, error) {
	if s == nil || s.generator == nil {
		return nil, errors.New("generator service is not configured")
	}

	result, err := s.generator.BuildGenerateSingleTaskResult(ctx, req.SourceType, content)
	if err != nil {
		return nil, err
	}

	return result.ResultJSON, nil
}

// BuildContinueTaskResult 基于续写追加正文构造结构化任务结果。
func (s *Service) BuildContinueTaskResult(ctx context.Context, userID uuid.UUID, blogID uuid.UUID, appendedContent string) ([]byte, error) {
	if s == nil || s.decomposition == nil {
		return nil, errors.New("decomposition service is not configured")
	}

	snapshot, err := s.decomposition.BuildContinueTaskResult(ctx, userID, blogID, appendedContent)
	if err != nil {
		return nil, err
	}

	return BuildContinueTaskResult(ContinueTaskResultInput{
		BlogID:          snapshot.BlogID,
		AppendedContent: snapshot.AppendedContent,
		FinalContent:    snapshot.FinalContent,
		EstimatedTokens: snapshot.EstimatedTokens,
		Usage: TaskResultUsage{
			EstimatedTokens:       snapshot.EstimatedTokens,
			PromptTokens:          snapshot.Usage.PromptTokens,
			CompletionTokens:      snapshot.Usage.CompletionTokens,
			PromptCacheHitTokens:  snapshot.Usage.PromptCacheHitTokens,
			PromptCacheMissTokens: snapshot.Usage.PromptCacheMissTokens,
		},
	})
}

// BuildGenerateSeriesTaskResult 提取系列生成完成后缓存的结构化任务结果。
func (s *Service) BuildGenerateSeriesTaskResult(ctx context.Context, req GenerateRequest) ([]byte, error) {
	if s == nil || s.decomposition == nil {
		return nil, errors.New("decomposition service is not configured")
	}

	parentID, err := uuid.Parse(req.ParentID)
	if err != nil {
		return nil, errors.New("invalid generation payload")
	}

	_ = ctx
	return s.decomposition.TakeGenerateSeriesTaskResult(parentID)
}

func (s *Service) Continue(bgCtx context.Context, userID uuid.UUID, blogID uuid.UUID, chunkChan chan<- string, errChan chan<- error) {
	s.decomposition.ContinueGeneration(bgCtx, userID, blogID, chunkChan, errChan)
}

func (s *Service) Polish(ctx context.Context, req PolishRequest, chunkChan chan<- string, errChan chan<- error) {
	s.generator.GeneratePolishDraftStream(ctx, req.Title, req.Content, chunkChan, errChan)
}

func (s *Service) AnalyzeStream(bgCtx context.Context, userID uuid.UUID, req GenerateRequest, progressChan chan<- string, errChan chan<- error) {
	scenarioMode := prompt.ScenarioMode(req.ScenarioMode)
	if req.SourceType == "file" {
		s.decomposition.AnalyzeFileStream(bgCtx, userID, req.SourceContent, scenarioMode, progressChan, errChan)
		return
	}
	s.decomposition.AnalyzeStream(bgCtx, userID, req.GitURL, req.SelectedModules, scenarioMode, progressChan, errChan)
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
