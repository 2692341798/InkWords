package projectanalysis

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"

	sharedprompt "inkwords-backend/shared/kernel/prompt"
	llm "inkwords-backend/shared/platform/llm"
	"inkwords-backend/shared/platform/parser"
	projectdomain "inkwords-backend/services/core-api/domain/project"
)

// LLMClient 是项目分析服务所需的 LLM 调用窄接口，仅暴露 GenerateJSONWithOptions 方法。
type LLMClient interface {
	GenerateJSONWithOptions(ctx context.Context, model string, messages []llm.Message, options llm.ChatOptions) (string, llm.CompletionUsage, error)
}

// Service 是 projectanalysis 应用服务，负责扫描仓库模块与生成博客大纲。
type Service struct {
	llm     LLMClient
	limiter *rate.Limiter
}

// NewService 创建 projectanalysis 应用服务实例。
func NewService(llmClient LLMClient) *Service {
	rpmLimit := 10000
	if v := os.Getenv("LLM_API_RPM_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rpmLimit = n
		}
	}
	limit := rate.Limit(float64(rpmLimit) / 60.0)

	return &Service{
		llm:     llmClient,
		limiter: rate.NewLimiter(limit, 1),
	}
}

// ScanProjectModules 克隆仓库（partial clone）或使用 GitHub API 列出顶级目录并返回模块卡片。
// 行为与 legacy internal/service.(*DecompositionService).ScanProjectModules 完全一致。
func (s *Service) ScanProjectModules(ctx context.Context, gitURL string) ([]projectdomain.ModuleCard, error) {
	var dirNames []string

	githubAPIBase := "https://api.github.com"
	if v := os.Getenv("GITHUB_API_BASE"); v != "" {
		githubAPIBase = strings.TrimRight(v, "/")
	}

	// 1. Try GitHub REST API first (fastest, no clone required)
	if owner, repo, ok := parser.ParseGithubOwnerRepo(gitURL); ok {
		apiURL := fmt.Sprintf("%s/repos/%s/%s/contents", githubAPIBase, owner, repo)
		client := &http.Client{Timeout: 10 * time.Second}
		req, _ := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			var contents []struct {
				Name string `json:"name"`
				Type string `json:"type"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&contents); err == nil {
				for _, item := range contents {
					if item.Type == "dir" {
						dirNames = append(dirNames, item.Name)
					}
				}
			}
			resp.Body.Close()
		} else if resp != nil {
			resp.Body.Close()
		}
	}

	var tempDir string
	// 2. Fallback to git partial clone if API failed or not a GitHub URL
	if len(dirNames) == 0 {
		var err error
		tempDir, err = os.MkdirTemp("", "inkwords-scan-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp dir: %w", err)
		}
		defer os.RemoveAll(tempDir)

		var stderr bytes.Buffer
		cmd := exec.Command("git", "-c", "http.postBuffer=1048576000", "-c", "http.maxRequestBuffer=100M", "-c", "core.compression=0", "-c", "http.lowSpeedLimit=1000", "-c", "http.lowSpeedTime=60", "clone", "--filter=blob:none", "--no-checkout", "--depth", "1", gitURL, tempDir)
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			stderr.Reset()
			cmd = exec.Command("git", "-c", "http.postBuffer=1048576000", "-c", "http.maxRequestBuffer=100M", "-c", "core.compression=0", "-c", "http.lowSpeedLimit=1000", "-c", "http.lowSpeedTime=60", "clone", "--no-checkout", "--depth", "1", gitURL, tempDir)
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				return nil, fmt.Errorf("failed to clone repository: %w, stderr: %s", err, stderr.String())
			}
		}

		cmdTree := exec.Command("git", "ls-tree", "-d", "--name-only", "HEAD")
		cmdTree.Dir = tempDir
		outBytes, err := cmdTree.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to run git ls-tree: %w", err)
		}

		dirNames = strings.Split(strings.TrimSpace(string(outBytes)), "\n")
	}

	// 3. Attempt to fetch README to generate intelligent descriptions
	var readmeContent string

	if owner, repo, ok := parser.ParseGithubOwnerRepo(gitURL); ok {
		readmeAPI := fmt.Sprintf("%s/repos/%s/%s/readme", githubAPIBase, owner, repo)
		client := &http.Client{Timeout: 10 * time.Second}
		req, _ := http.NewRequestWithContext(ctx, "GET", readmeAPI, nil)
		req.Header.Set("Accept", "application/vnd.github.v3.raw")
		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			readmeContent = buf.String()
			resp.Body.Close()
		} else if resp != nil {
			resp.Body.Close()
		}
	}

	if readmeContent == "" && tempDir != "" {
		for _, name := range []string{"README.md", "README", "readme.md", "Readme.md"} {
			cmd := exec.Command("git", "show", "HEAD:"+name)
			cmd.Dir = tempDir
			if out, err := cmd.Output(); err == nil {
				readmeContent = string(out)
				break
			}
		}
	}

	var dirDescriptions map[string]string

	if len(dirNames) > 0 && readmeContent != "" {
		if len(readmeContent) > 20000 {
			readmeContent = readmeContent[:20000]
		}

		dirNamesJson, _ := json.Marshal(dirNames)
		prompt := fmt.Sprintf("你是一个高级技术架构师。以下是一个项目的根目录列表：\n%s\n\n请结合以下项目的 README 内容，为每个目录提供一句简短的中文描述（10-20个字左右），说明该目录的用途。如果 README 中未提及，请根据开源项目常见命名规范进行推测。\n\n输出格式必须是一个 JSON 对象，键为目录名，值为描述。不要包含任何 markdown 代码块标记，纯 JSON 输出。\n\nREADME 内容：\n%s", string(dirNamesJson), readmeContent)

		messages := []llm.Message{
			{Role: "user", Content: prompt},
		}

		modelStr := "deepseek-v4-flash"
		if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
			modelStr = envModel
		}

		ctxTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		if err := s.limiter.Wait(ctxTimeout); err == nil {
			if jsonStr, _, err := s.llm.GenerateJSONWithOptions(ctxTimeout, modelStr, messages, llm.LightweightChatOptions("", 1000)); err == nil {
				jsonStr = strings.TrimPrefix(strings.TrimSpace(jsonStr), "```json")
				jsonStr = strings.TrimPrefix(jsonStr, "```")
				jsonStr = strings.TrimSuffix(jsonStr, "```")
				jsonStr = strings.TrimSpace(jsonStr)

				_ = json.Unmarshal([]byte(jsonStr), &dirDescriptions)
			}
		}
	}

	if dirDescriptions == nil {
		dirDescriptions = make(map[string]string)
	}

	var modules []projectdomain.ModuleCard
	ignoredDirs := map[string]bool{
		".git": true, "node_modules": true, "vendor": true, "dist": true,
		"build": true, "docs": true, "assets": true, "public": true,
		"tests": true, "test": true, ".github": true, ".vscode": true,
	}

	for _, dirName := range dirNames {
		dirName = strings.TrimSpace(dirName)
		if dirName == "" || ignoredDirs[dirName] || strings.HasPrefix(dirName, ".") {
			continue
		}

		desc := dirDescriptions[dirName]
		if desc == "" {
			desc = "代码目录模块 (点击解析后查看大纲)"
		}

		modules = append(modules, projectdomain.ModuleCard{
			Path:        dirName,
			Name:        dirName,
			Description: desc,
		})
	}

	return modules, nil
}

// GenerateOutline 根据源码内容生成系列博客大纲。
// 行为与 legacy internal/service.(*DecompositionService).GenerateOutline(existingParent=nil, existingChildren=nil) 完全一致。
func (s *Service) GenerateOutline(ctx context.Context, sourceContent string, scenarioMode sharedprompt.ScenarioMode) (projectdomain.OutlineResult, error) {
	runes := []rune(sourceContent)
	if len(runes) > 15000000 {
		sourceContent = string(runes[:15000000]) + "\n\n... [Content Truncated due to length limits] ..."
	}

	if !scenarioMode.IsValid() {
		scenarioMode = sharedprompt.ScenarioModeEbookInterpretation
	}

	profile := normalizePromptProfile(sharedprompt.PromptProfile{}, scenarioMode)
	systemRole, instruction := outlinePromptForProfile(scenarioMode, profile)

	systemLabel := "项目文本内容如下：\n"
	if scenarioMode == sharedprompt.ScenarioModeEbookInterpretation {
		systemLabel = "以下是原文内容：\n"
	}

	messages := []llm.Message{
		{Role: "system", Content: systemRole + "\n\n" + systemLabel + sourceContent},
		{Role: "user", Content: instruction},
	}

	modelName := "deepseek-v4-flash"
	if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
		modelName = envModel
	}

	options := llm.DefaultChatOptions()
	options.MaxTokens = 6000
	content, _, err := s.llm.GenerateJSONWithOptions(ctx, modelName, messages, options)
	if err != nil {
		return projectdomain.OutlineResult{}, fmt.Errorf("llm generation failed: %w", err)
	}

	content = strings.TrimPrefix(strings.TrimSpace(content), "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var outline outlineResult
	if err := json.Unmarshal([]byte(content), &outline); err != nil {
		return projectdomain.OutlineResult{}, fmt.Errorf("failed to unmarshal llm output: %w, output: %s", err, content)
	}

	chapters := make([]projectdomain.Chapter, 0, len(outline.Chapters))
	for _, ch := range outline.Chapters {
		chapters = append(chapters, projectdomain.Chapter{
			ID:      ch.ID,
			Title:   ch.Title,
			Summary: ch.Summary,
			Sort:    ch.Sort,
			Files:   ch.Files,
			Action:  ch.Action,
		})
	}
	return projectdomain.OutlineResult{
		SeriesTitle: outline.SeriesTitle,
		Chapters:    chapters,
		ParentID:    outline.ParentID,
	}, nil
}

// outlineResult 是 LLM 输出的原始 JSON 大纲结构，用于反序列化。
type outlineResult struct {
	SeriesTitle string       `json:"series_title"`
	Chapters    []rawChapter `json:"chapters"`
	ParentID    string       `json:"parent_id,omitempty"`
}

type rawChapter struct {
	ID      string   `json:"id,omitempty"`
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Sort    int      `json:"sort"`
	Files   []string `json:"files"`
	Action  string   `json:"action,omitempty"`
}

// outlinePromptForProfile 根据场景模式和 profile 组装 system role 与用户指令。
func outlinePromptForProfile(mode sharedprompt.ScenarioMode, profile sharedprompt.PromptProfile) (string, string) {
	return profile.SystemRole, strings.TrimSpace(strings.Join([]string{
		profile.AnalyzeRequirements,
		outlineBaseInstruction(mode),
		"场景约束：\n" + outlineScenarioHint(mode),
	}, "\n\n"))
}

// outlineBaseInstruction 返回不同场景下的大纲生成基础指令。
func outlineBaseInstruction(mode sharedprompt.ScenarioMode) string {
	if mode == sharedprompt.ScenarioModeEbookInterpretation {
		return `前面提供的是一本书或长篇文献的内容，请按原文自然篇章结构生成一个系列解读大纲。
我的核心要求是：按原文的章节或主题单元逐章拆分，每章聚焦该篇的核心思想与原文精义，而非现代应用或技术映射。

请根据文本的章节数量充分规划：
- 每一个独立篇章或主题段落都应至少对应一篇解读
- 每篇解读聚焦该篇章的历史背景、核心观点和代表性原文摘录
- 不要将内容强行映射到现代商业、技术或管理场景`
	}

	return `请评估前面提供的项目文本，并生成一个系列博客的大纲。
对于大型项目、源码仓库或复杂内容，必须拆分为系列博客。
我的核心要求是：对于每个核心模块、业务逻辑或重要架构层，**都必须对应至少一篇博客进行详细说明**。不要担心生成的篇数过多！

请根据提供的项目文件数量和内容复杂度，充分且详细地规划章节数量：
- 对于普通项目，不要吝啬章节数，确保每一个独立的功能点、数据流环节、配置模块等都有专属的文章解析。
- 对于特别庞大的框架源码（如 FFmpeg 等），请大胆拆分出数十篇详细章节，做到对核心源码文件的全面覆盖。
- 每一篇博客只聚焦于**一个具体的核心技术点或模块**，并详细说明其原理与实现。`
}

// outlineScenarioHint 根据场景模式返回对应的章节拆分约束。
func outlineScenarioHint(mode sharedprompt.ScenarioMode) string {
	switch mode {
	case sharedprompt.ScenarioModeOpenBookExamReview:
		return "请按考点、题型、实验步骤或速查结构拆分章节，优先帮助开卷考试快速定位。"
	case sharedprompt.ScenarioModeBeginnerWalkthrough:
		return "请按学习路径拆分章节，优先覆盖环境准备、目录结构、关键主链路和常见排错。"
	default:
		return "请按原文自身篇章结构与主题脉络拆分章节，只做文本解读，不要将内容映射到现代商业、技术或管理场景。"
	}
}

// normalizePromptProfile 将空 profile 按场景回退为兜底值，行为与 legacy 完全一致。
func normalizePromptProfile(profile sharedprompt.PromptProfile, scenarioMode sharedprompt.ScenarioMode) sharedprompt.PromptProfile {
	if !scenarioMode.IsValid() {
		scenarioMode = sharedprompt.ScenarioModeEbookInterpretation
	}
	if profile.Key == "" {
		return sharedprompt.FallbackPromptProfileForScenario(scenarioMode)
	}

	return sharedprompt.ResolvePromptProfileKey(string(profile.Key), scenarioMode)
}

// newResolvedPromptProfile 从 PromptProfile 构造 ResolvedPromptProfile，行为与 legacy 完全一致。
func newResolvedPromptProfile(profile sharedprompt.PromptProfile, documentKind string, reason string) sharedprompt.ResolvedPromptProfile {
	return sharedprompt.ResolvedPromptProfile{
		Key:          profile.Key,
		DisplayName:  profile.DisplayName,
		DocumentKind: documentKind,
		Reason:       reason,
	}
}
