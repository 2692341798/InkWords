package projectanalysis_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sharedprompt "inkwords-backend/shared/kernel/prompt"
	llm "inkwords-backend/shared/platform/llm"

	"inkwords-backend/services/core-api/app/projectanalysis"
)

// fakeLLMClient 是一个确定性测试桩，记录最后一次调用参数并通过 JSON 响应模拟 LLM 行为。
type fakeLLMClient struct {
	lastMessages []llm.Message
	lastModel    string
	lastOptions  llm.ChatOptions

	jsonResponse string
	err          error
}

func (f *fakeLLMClient) GenerateJSONWithOptions(ctx context.Context, model string, messages []llm.Message, options llm.ChatOptions) (string, llm.CompletionUsage, error) {
	f.lastModel = model
	f.lastMessages = append([]llm.Message(nil), messages...)
	f.lastOptions = options

	select {
	case <-ctx.Done():
		return "", llm.CompletionUsage{}, ctx.Err()
	default:
	}

	if f.err != nil {
		return "", llm.CompletionUsage{}, f.err
	}
	return f.jsonResponse, llm.CompletionUsage{}, nil
}

// githubStubServer 返回一个 httptest.Server，模拟 GitHub REST API 响应。
func githubStubServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/contents") && strings.Contains(r.URL.Path, "/repos/"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{
				{"name": "src", "type": "dir"},
				{"name": "pkg", "type": "dir"},
				{"name": "cmd", "type": "dir"},
				{"name": "README.md", "type": "file"},
				{"name": ".github", "type": "dir"},
				{"name": "docs", "type": "dir"},
			})
		case strings.HasSuffix(r.URL.Path, "/readme") && strings.Contains(r.URL.Path, "/repos/"):
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("# Test Repo\n\nsrc: core source code\npkg: public library packages\ncmd: command line tools"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestScanProjectModules_WithGitHubAPIMock(t *testing.T) {
	server := githubStubServer(t)
	defer server.Close()

	t.Setenv("GITHUB_API_BASE", server.URL)

	fakeLLM := &fakeLLMClient{
		jsonResponse: `{"src": "核心源码目录", "pkg": "公共库包", "cmd": "命令行工具"}`,
	}

	svc := projectanalysis.NewService(fakeLLM)

	gitURL := "https://github.com/testowner/testrepo"
	modules, err := svc.ScanProjectModules(context.Background(), gitURL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(modules) < 1 {
		t.Fatalf("expected at least 1 module, got %d", len(modules))
	}

	foundSrc := false
	for _, m := range modules {
		if m.Path == "src" {
			foundSrc = true
			if m.Name != "src" {
				t.Errorf("expected module name 'src', got %q", m.Name)
			}
			if m.Description == "" {
				t.Errorf("module 'src' has empty description")
			}
		}
		if strings.HasPrefix(m.Path, ".") {
			t.Errorf("hidden directory %q should be filtered", m.Path)
		}
	}
	if !foundSrc {
		t.Error("expected 'src' directory to be included in module cards")
	}
}

//nolint:gosec,noctx
func TestScanProjectModules_NoGitHubFallback(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	fakeLLM := &fakeLLMClient{
		jsonResponse: `{}`,
	}

	svc := projectanalysis.NewService(fakeLLM)

	tempDir := t.TempDir()
	initCmd := exec.Command("git", "init", tempDir)
	if err := initCmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	for key, value := range map[string]string{
		"user.name":  "InkWords Tests",
		"user.email": "tests@inkwords.local",
	} {
		configCmd := exec.Command("git", "-C", tempDir, "config", key, value)
		if err := configCmd.Run(); err != nil {
			t.Fatalf("git config %s failed: %v", key, err)
		}
	}

	mylibDir := filepath.Join(tempDir, "mylib")
	if err := os.MkdirAll(mylibDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	// git 不跟踪空目录，需要在目录内放置一个文件
	if err := os.WriteFile(filepath.Join(mylibDir, ".gitkeep"), []byte{}, 0644); err != nil {
		t.Fatalf("write .gitkeep failed: %v", err)
	}

	commitCmd := exec.Command("git", "-C", tempDir, "add", "-A")
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	commitCmd = exec.Command("git", "-C", tempDir, "commit", "-m", "init", "--allow-empty")
	if err := commitCmd.Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	modules, err := svc.ScanProjectModules(context.Background(), tempDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundMylib := false
	for _, m := range modules {
		if m.Path == "mylib" {
			foundMylib = true
		}
	}
	if !foundMylib {
		t.Error("expected 'mylib' directory in module cards from local git repo")
	}
}

func TestScanProjectModules_LLMErrorStillReturnsModules(t *testing.T) {
	server := githubStubServer(t)
	defer server.Close()

	t.Setenv("GITHUB_API_BASE", server.URL)

	fakeLLM := &fakeLLMClient{
		err:          &llm.APIError{StatusCode: http.StatusTooManyRequests, Body: "rate limited"},
		jsonResponse: `{}`,
	}

	svc := projectanalysis.NewService(fakeLLM)

	modules, err := svc.ScanProjectModules(context.Background(), "https://github.com/testowner/testrepo")
	if err != nil {
		t.Fatalf("scan should not fail on LLM error: %v", err)
	}

	if len(modules) < 1 {
		t.Fatalf("expected at least 1 module even when LLM fails, got %d", len(modules))
	}

	for _, m := range modules {
		if m.Description == "" {
			t.Errorf("module %q has empty description, expected fallback text", m.Path)
		}
	}
}

func TestScanProjectModules_InvalidGitURL(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	fakeLLM := &fakeLLMClient{
		jsonResponse: `{}`,
	}

	svc := projectanalysis.NewService(fakeLLM)

	_, err := svc.ScanProjectModules(context.Background(), "not-a-valid-url-or-repo")
	if err == nil {
		t.Fatal("expected error for invalid git URL, got nil")
	}
}

func TestGenerateOutline_Success(t *testing.T) {
	outlineResp := `{
		"series_title": "Test Project 源码解析系列",
		"chapters": [
			{"id": "", "title": "项目架构概览", "summary": "整体架构介绍", "sort": 1, "files": ["src/main.go"], "action": "new"},
			{"id": "", "title": "核心模块解析", "summary": "核心模块详解", "sort": 2, "files": ["src/core/"], "action": "new"}
		],
		"parent_id": ""
	}`

	fakeLLM := &fakeLLMClient{
		jsonResponse: outlineResp,
	}

	svc := projectanalysis.NewService(fakeLLM)

	result, err := svc.GenerateOutline(context.Background(), "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}", sharedprompt.ScenarioModeBeginnerWalkthrough)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.SeriesTitle != "Test Project 源码解析系列" {
		t.Errorf("expected series title 'Test Project 源码解析系列', got %q", result.SeriesTitle)
	}
	if len(result.Chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(result.Chapters))
	}
	if result.Chapters[0].Title != "项目架构概览" {
		t.Errorf("expected first chapter '项目架构概览', got %q", result.Chapters[0].Title)
	}
	if result.Chapters[0].Sort != 1 {
		t.Errorf("expected sort 1, got %d", result.Chapters[0].Sort)
	}

	if len(fakeLLM.lastMessages) != 2 {
		t.Fatalf("expected 2 messages (system + user), got %d", len(fakeLLM.lastMessages))
	}
	if fakeLLM.lastMessages[0].Role != "system" {
		t.Errorf("expected system role, got %q", fakeLLM.lastMessages[0].Role)
	}
	if fakeLLM.lastMessages[1].Role != "user" {
		t.Errorf("expected user role, got %q", fakeLLM.lastMessages[1].Role)
	}
	if fakeLLM.lastOptions.MaxTokens != 6000 {
		t.Errorf("expected MaxTokens 6000, got %d", fakeLLM.lastOptions.MaxTokens)
	}
}

func TestGenerateOutline_LLMError(t *testing.T) {
	fakeLLM := &fakeLLMClient{
		err: &llm.APIError{StatusCode: http.StatusInternalServerError, Body: "internal error"},
	}

	svc := projectanalysis.NewService(fakeLLM)

	_, err := svc.GenerateOutline(context.Background(), "some source", sharedprompt.ScenarioModeBeginnerWalkthrough)
	if err == nil {
		t.Fatal("expected error from LLM failure, got nil")
	}
	if !strings.Contains(err.Error(), "llm generation failed") {
		t.Errorf("expected 'llm generation failed' in error, got: %v", err)
	}
}

func TestGenerateOutline_ContextCanceled(t *testing.T) {
	fakeLLM := &fakeLLMClient{
		jsonResponse: `{"series_title":"x","chapters":[],"parent_id":""}`,
	}

	svc := projectanalysis.NewService(fakeLLM)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	time.Sleep(2 * time.Millisecond)

	_, err := svc.GenerateOutline(ctx, "some source", sharedprompt.ScenarioModeBeginnerWalkthrough)
	if err == nil {
		t.Fatal("expected error from canceled context, got nil")
	}
	if !strings.Contains(err.Error(), "llm generation failed") && !strings.Contains(err.Error(), "context") {
		t.Errorf("expected context-related error, got: %v", err)
	}
}

func TestGenerateOutline_InvalidJSON(t *testing.T) {
	fakeLLM := &fakeLLMClient{
		jsonResponse: "not json at all {",
	}

	svc := projectanalysis.NewService(fakeLLM)

	_, err := svc.GenerateOutline(context.Background(), "some source", sharedprompt.ScenarioModeBeginnerWalkthrough)
	if err == nil {
		t.Fatal("expected unmarshal error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to unmarshal") {
		t.Errorf("expected unmarshal error, got: %v", err)
	}
}

func TestGenerateOutline_EmptyJSON(t *testing.T) {
	fakeLLM := &fakeLLMClient{
		jsonResponse: `{}`,
	}

	svc := projectanalysis.NewService(fakeLLM)

	result, err := svc.GenerateOutline(context.Background(), "some source", sharedprompt.ScenarioModeBeginnerWalkthrough)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SeriesTitle != "" {
		t.Errorf("expected empty series title, got %q", result.SeriesTitle)
	}
	if len(result.Chapters) != 0 {
		t.Errorf("expected 0 chapters, got %d", len(result.Chapters))
	}
}

func TestGenerateOutline_ContentTruncation(t *testing.T) {
	fakeLLM := &fakeLLMClient{
		jsonResponse: `{"series_title":"test","chapters":[],"parent_id":""}`,
	}

	svc := projectanalysis.NewService(fakeLLM)

	largeContent := strings.Repeat("a", 15000001)

	_, err := svc.GenerateOutline(context.Background(), largeContent, sharedprompt.ScenarioModeBeginnerWalkthrough)
	if err != nil {
		t.Fatalf("unexpected error with large content: %v", err)
	}

	if len(fakeLLM.lastMessages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(fakeLLM.lastMessages))
	}
	systemContent := fakeLLM.lastMessages[0].Content
	if !strings.Contains(systemContent, "[Content Truncated") {
		t.Error("expected truncation marker in system message content")
	}
}

func TestGenerateOutline_Scenarios(t *testing.T) {
	tests := []struct {
		name  string
		mode  sharedprompt.ScenarioMode
		label string
	}{
		{name: "BeginnerWalkthrough", mode: sharedprompt.ScenarioModeBeginnerWalkthrough, label: "项目文本内容"},
		{name: "EbookInterpretation", mode: sharedprompt.ScenarioModeEbookInterpretation, label: "原文内容"},
		{name: "OpenBookExamReview", mode: sharedprompt.ScenarioModeOpenBookExamReview, label: "项目文本内容"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeLLM := &fakeLLMClient{
				jsonResponse: `{"series_title":"test","chapters":[],"parent_id":""}`,
			}
			svc := projectanalysis.NewService(fakeLLM)

			_, err := svc.GenerateOutline(context.Background(), "content", tt.mode)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.Contains(fakeLLM.lastMessages[0].Content, tt.label) {
				t.Errorf("expected system label %q in message, got: %s", tt.label, fakeLLM.lastMessages[0].Content[:200])
			}
		})
	}
}

func TestGenerateOutline_InvalidScenarioFallsBack(t *testing.T) {
	fakeLLM := &fakeLLMClient{
		jsonResponse: `{"series_title":"test","chapters":[],"parent_id":""}`,
	}
	svc := projectanalysis.NewService(fakeLLM)

	_, err := svc.GenerateOutline(context.Background(), "content", sharedprompt.ScenarioMode("invalid"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(fakeLLM.lastMessages[0].Content, "原文内容") {
		t.Error("expected fallback to EbookInterpretation system label")
	}
}

func TestScanProjectModules_ContextCanceledWithGitHubAPI(t *testing.T) {
	server := githubStubServer(t)
	defer server.Close()

	t.Setenv("GITHUB_API_BASE", server.URL)

	fakeLLM := &fakeLLMClient{
		jsonResponse: `{}`,
	}

	svc := projectanalysis.NewService(fakeLLM)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := svc.ScanProjectModules(ctx, "https://github.com/testowner/testrepo")
	if err == nil {
		t.Fatal("expected error when context is canceled and fallback fails")
	}
}
