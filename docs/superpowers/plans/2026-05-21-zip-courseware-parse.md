# ZIP Courseware Parse Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 InkWords 新增 ZIP 课件包上传解析能力，自动筛选并去重 ZIP 内有效文档/代码文本，聚合为统一 `source_content` 后继续复用现有分析链路。

**Architecture:** 保持前端上传流程基本不变，只在上传入口和解析提示层做最小修改。后端在 `project parse` 链路中新增 ZIP 专用解析器，完成临时解压、白名单过滤、文本提取、去重、顺序聚合和摘要返回，同时保留现有单文件解析路径不回归。

**Tech Stack:** Go 1.21 + Gin + existing parser package + React 18 + Vite + Vitest

---

## File Structure Map

**Backend parser**
- Create: `backend/internal/infra/parser/archive_parser.go`
  - 负责 ZIP 临时落盘、解压、安全校验、候选文件筛选、文本提取、去重、顺序聚合和 `archive_summary` 统计。
- Create: `backend/internal/infra/parser/archive_parser_test.go`
  - 覆盖 ZIP 混合文件、完全重复、路径穿越、无有效文件等核心场景。
- Modify: `backend/internal/infra/parser/doc_parser.go`
  - 仅提取可复用的文本文件后缀判断或公共帮助函数，避免 ZIP 解析器重复维护同一份规则。

**Backend project domain**
- Modify: `backend/internal/domain/project/dto.go`
  - 新增解析结果 DTO，承载 `source_content` 和可选 `archive_summary`。
- Modify: `backend/internal/domain/project/service.go`
  - 将 `Parse` 返回值从纯字符串扩展为结构化结果，并在 ZIP 场景调度 `ArchiveParser`。
- Modify: `backend/internal/domain/project/handler.go`
  - 返回 ZIP 解析摘要，同时保持非 ZIP 响应兼容。
- Create: `backend/internal/domain/project/handler_parse_test.go`
  - 覆盖 `/parse` 正常上传 ZIP、无有效文本 ZIP 和普通文件不带摘要的响应差异。
- Modify: `backend/internal/transport/http/v1/api/project.go`
  - 完成新增解析器依赖注入。

**Frontend upload flow**
- Modify: `frontend/src/components/generator/GeneratorInput.tsx`
  - 接受 `.zip` 上传，并把说明文案改为中文的“单文件或 ZIP 课件包”。
- Modify: `frontend/src/hooks/generator/fileParserUtils.ts`
  - 新增 `archive_summary` 读取方法与响应类型。
- Modify: `frontend/src/hooks/generator/fileParserUtils.test.ts`
  - 覆盖新响应结构解析。
- Modify: `frontend/src/hooks/generator/useFileParser.ts`
  - 上传成功后在分析历史里追加 ZIP 解析摘要，普通文件链路保持原状。

**Docs sync**
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `.trae/documents/InkWords_PRD.md`
- Modify: `README.md`
  - 记录 ZIP 课件解析能力、返回结构和限制约束。

### Task 1: Build the backend ZIP parser

**Files:**
- Create: `backend/internal/infra/parser/archive_parser.go`
- Create: `backend/internal/infra/parser/archive_parser_test.go`
- Modify: `backend/internal/infra/parser/doc_parser.go`
- Test: `backend/internal/infra/parser/archive_parser_test.go`

- [ ] **Step 1: Write the failing ZIP parser tests**

```go
func TestArchiveParser_ParseArchive_KeepsUsefulFilesAndDeduplicates(t *testing.T) {
	parser := NewArchiveParser(NewDocParser())
	archive := buildZipArchive(t, map[string]string{
		"01-intro.md":         "# 第一课\n课程目标",
		"02-demo.txt":         "示例代码说明",
		"copies/02-demo.txt":  "示例代码说明",
		"src/main.go":         "package main\nfunc main() {}\n",
		"assets/logo.png":     "not-text",
	})

	result, err := parser.ParseArchive(bytes.NewReader(archive), "courseware.zip")
	require.NoError(t, err)
	assert.Contains(t, result.SourceContent, "--- 文件: 01-intro.md ---")
	assert.Contains(t, result.SourceContent, "--- 文件: 02-demo.txt ---")
	assert.Contains(t, result.SourceContent, "--- 文件: src/main.go ---")
	assert.Equal(t, 5, result.ArchiveSummary.TotalFiles)
	assert.Equal(t, 4, result.ArchiveSummary.SupportedFiles)
	assert.Equal(t, 3, result.ArchiveSummary.KeptFiles)
	assert.Equal(t, 1, result.ArchiveSummary.DuplicateFiles)
	assert.Equal(t, 1, result.ArchiveSummary.IgnoredFiles)
}

func TestArchiveParser_ParseArchive_RejectsPathTraversal(t *testing.T) {
	parser := NewArchiveParser(NewDocParser())
	archive := buildZipArchive(t, map[string]string{
		"../../escape.md": "# bad",
	})

	_, err := parser.ParseArchive(bytes.NewReader(archive), "danger.zip")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "非法压缩包路径")
}

func TestArchiveParser_ParseArchive_ReturnsErrorWhenNoUsefulFiles(t *testing.T) {
	parser := NewArchiveParser(NewDocParser())
	archive := buildZipArchive(t, map[string]string{
		"notes/logo.png": "binary",
		"empty.txt":      "\n\n",
	})

	_, err := parser.ParseArchive(bytes.NewReader(archive), "empty.zip")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "压缩包中没有可解析的文本文件")
}
```

- [ ] **Step 2: Run the parser tests to verify they fail**

Run: `go test ./internal/infra/parser -run ArchiveParser -v`  
Expected: FAIL with `undefined: NewArchiveParser` and missing ZIP helper functions.

- [ ] **Step 3: Implement the ZIP parser with safe extraction and summary stats**

```go
type ArchiveSummary struct {
	TotalFiles     int      `json:"total_files"`
	SupportedFiles int      `json:"supported_files"`
	KeptFiles      int      `json:"kept_files"`
	DuplicateFiles int      `json:"duplicate_files"`
	IgnoredFiles   int      `json:"ignored_files"`
	FailedFiles    int      `json:"failed_files"`
	KeptPaths      []string `json:"kept_paths"`
}

type ParsedSource struct {
	SourceContent  string          `json:"source_content"`
	ArchiveSummary *ArchiveSummary `json:"archive_summary,omitempty"`
}

type ArchiveParser struct {
	docParser *DocParser
}

func NewArchiveParser(docParser *DocParser) *ArchiveParser {
	return &ArchiveParser{docParser: docParser}
}

func (p *ArchiveParser) ParseArchive(src io.Reader, filename string) (ParsedSource, error) {
	if strings.ToLower(filepath.Ext(filename)) != ".zip" {
		return ParsedSource{}, fmt.Errorf("unsupported archive extension: %s", filepath.Ext(filename))
	}

	zipPath, cleanupFile, err := writeArchiveTempFile(src)
	if err != nil {
		return ParsedSource{}, err
	}
	defer cleanupFile()

	extractDir, cleanupDir, err := extractArchiveToTempDir(zipPath)
	if err != nil {
		return ParsedSource{}, err
	}
	defer cleanupDir()

	files, err := collectArchiveCandidates(extractDir)
	if err != nil {
		return ParsedSource{}, err
	}

	sort.SliceStable(files, func(i, j int) bool {
		return naturalLess(files[i].ArchivePath, files[j].ArchivePath)
	})

	summary := &ArchiveSummary{TotalFiles: len(files)}
	seen := map[string]struct{}{}
	var parts []string

	for _, candidate := range files {
		if !isSupportedArchiveTextFile(candidate.ArchivePath) {
			summary.IgnoredFiles++
			continue
		}
		summary.SupportedFiles++

		text, err := p.parseCandidate(candidate)
		if err != nil {
			summary.FailedFiles++
			continue
		}

		normalized := normalizeArchiveText(text)
		if utf8.RuneCountInString(normalized) < 20 {
			summary.IgnoredFiles++
			continue
		}

		fingerprint := sha256.Sum256([]byte(normalized))
		key := hex.EncodeToString(fingerprint[:])
		if _, ok := seen[key]; ok {
			summary.DuplicateFiles++
			continue
		}

		seen[key] = struct{}{}
		summary.KeptFiles++
		summary.KeptPaths = append(summary.KeptPaths, candidate.ArchivePath)
		parts = append(parts, fmt.Sprintf("--- 文件: %s ---\n%s", candidate.ArchivePath, normalized))
	}

	if len(parts) == 0 {
		return ParsedSource{}, fmt.Errorf("压缩包中没有可解析的文本文件")
	}

	return ParsedSource{
		SourceContent:  strings.Join(parts, "\n\n"),
		ArchiveSummary: summary,
	}, nil
}
```

- [ ] **Step 4: Add shared helpers in `doc_parser.go` instead of duplicating extension logic**

```go
var plainTextExtensions = map[string]bool{
	".md":       true,
	".markdown": true,
	".txt":      true,
}

func isPlainTextExtension(ext string) bool {
	return plainTextExtensions[strings.ToLower(ext)]
}
```

- [ ] **Step 5: Run the parser tests to verify they pass**

Run: `go test ./internal/infra/parser -run ArchiveParser -v`  
Expected: PASS for ZIP aggregation, deduplication, and path traversal protection.

- [ ] **Step 6: Commit the parser slice**

```bash
git add backend/internal/infra/parser/archive_parser.go \
  backend/internal/infra/parser/archive_parser_test.go \
  backend/internal/infra/parser/doc_parser.go
git commit -m "feat(parser): add zip courseware archive parser"
```

### Task 2: Wire ZIP parsing into the project parse API

**Files:**
- Modify: `backend/internal/domain/project/dto.go`
- Modify: `backend/internal/domain/project/service.go`
- Modify: `backend/internal/domain/project/handler.go`
- Modify: `backend/internal/transport/http/v1/api/project.go`
- Create: `backend/internal/domain/project/handler_parse_test.go`
- Test: `backend/internal/domain/project/handler_parse_test.go`

- [ ] **Step 1: Write the failing handler tests for ZIP and non-ZIP responses**

```go
func TestHandler_Parse_ReturnsArchiveSummaryForZip(t *testing.T) {
	service := &stubProjectService{
		parseResult: ParseResult{
			SourceContent: "merged content",
			ArchiveSummary: &parser.ArchiveSummary{
				TotalFiles: 3,
				KeptFiles:  2,
			},
		},
	}
	handler := NewHandler(service)

	w := performMultipartParseRequest(t, handler, "courseware.zip", []byte("fake zip"))
	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{
		"code": 200,
		"message": "success",
		"data": {
			"source_content": "merged content",
			"archive_summary": {
				"total_files": 3,
				"kept_files": 2
			}
		}
	}`, w.Body.String())
}

func TestHandler_Parse_OmitsArchiveSummaryForNormalFile(t *testing.T) {
	service := &stubProjectService{
		parseResult: ParseResult{SourceContent: "plain content"},
	}
	handler := NewHandler(service)

	w := performMultipartParseRequest(t, handler, "lesson.md", []byte("# title"))
	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "archive_summary")
}
```

- [ ] **Step 2: Run the handler tests to verify they fail**

Run: `go test ./internal/domain/project -run Parse -v`  
Expected: FAIL because `ParseResult` and `archive_summary` response fields do not exist yet.

- [ ] **Step 3: Expand the project parse DTO and service return type**

```go
type ParseResult struct {
	SourceContent  string                 `json:"source_content"`
	ArchiveSummary *parser.ArchiveSummary `json:"archive_summary,omitempty"`
}

type ArchiveParser interface {
	ParseArchive(src io.Reader, filename string) (parser.ParsedSource, error)
}

type Service struct {
	decomposition *service.DecompositionService
	gitFetcher    *parser.GitFetcher
	docParser     *parser.DocParser
	archiveParser *parser.ArchiveParser
	userService   *service.UserService
}

func (s *Service) Parse(file io.Reader, filename string) (ParseResult, error) {
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
```

- [ ] **Step 4: Return the new response shape from the handler and inject the parser in the API factory**

```go
parseResult, err := h.service.Parse(file, header.Filename)
if err != nil {
	c.JSON(http.StatusInternalServerError, gin.H{
		"code":    http.StatusInternalServerError,
		"message": "解析文件失败: " + err.Error(),
		"data":    nil,
	})
	return
}

response := gin.H{
	"source_content": parseResult.SourceContent,
}
if parseResult.ArchiveSummary != nil {
	response["archive_summary"] = parseResult.ArchiveSummary
}

c.JSON(http.StatusOK, gin.H{
	"code":    http.StatusOK,
	"message": "success",
	"data":    response,
})
```

- [ ] **Step 5: Run the handler tests to verify they pass**

Run: `go test ./internal/domain/project -run Parse -v`  
Expected: PASS for ZIP response metadata and normal file backward compatibility.

- [ ] **Step 6: Run a focused backend regression pass**

Run: `go test ./internal/infra/parser ./internal/domain/project ./internal/domain/stream -v`  
Expected: PASS with no regressions in file-source parsing.

- [ ] **Step 7: Commit the API slice**

```bash
git add backend/internal/domain/project/dto.go \
  backend/internal/domain/project/service.go \
  backend/internal/domain/project/handler.go \
  backend/internal/domain/project/handler_parse_test.go \
  backend/internal/transport/http/v1/api/project.go
git commit -m "feat(project): return zip archive parse summary"
```

### Task 3: Update the frontend upload flow for ZIP input

**Files:**
- Modify: `frontend/src/components/generator/GeneratorInput.tsx`
- Modify: `frontend/src/hooks/generator/fileParserUtils.ts`
- Modify: `frontend/src/hooks/generator/fileParserUtils.test.ts`
- Modify: `frontend/src/hooks/generator/useFileParser.ts`
- Test: `frontend/src/hooks/generator/fileParserUtils.test.ts`

- [ ] **Step 1: Write the failing frontend utility test for archive summary parsing**

```ts
it('reads archive summary from the backend data wrapper', () => {
  expect(
    extractArchiveSummary({
      data: {
        archive_summary: {
          total_files: 8,
          kept_files: 3,
          duplicate_files: 2,
          ignored_files: 2,
          failed_files: 1,
        },
      },
    }),
  ).toEqual({
    total_files: 8,
    kept_files: 3,
    duplicate_files: 2,
    ignored_files: 2,
    failed_files: 1,
  })
})
```

- [ ] **Step 2: Run the frontend test to verify it fails**

Run: `cd frontend && npm test -- src/hooks/generator/fileParserUtils.test.ts`  
Expected: FAIL with `extractArchiveSummary is not defined`.

- [ ] **Step 3: Extend the file parser utilities and show ZIP summary in the analysis history**

```ts
export interface ArchiveSummary {
  total_files: number
  supported_files?: number
  kept_files: number
  duplicate_files: number
  ignored_files: number
  failed_files: number
  kept_paths?: string[]
}

interface ParseFileResponse {
  content?: string
  data?: {
    source_content?: string
    archive_summary?: ArchiveSummary
  }
}

export function extractArchiveSummary(response: ParseFileResponse): ArchiveSummary | undefined {
  return response.data?.archive_summary
}
```

```ts
const summary = extractArchiveSummary(data)
if (summary) {
  store.appendAnalysisHistory({
    message: `压缩包共扫描 ${summary.total_files} 个文件，保留 ${summary.kept_files} 个，去重 ${summary.duplicate_files} 个，忽略 ${summary.ignored_files} 个，失败 ${summary.failed_files} 个`,
    status: 'parsed',
  })
}
```

- [ ] **Step 4: Update the upload component to accept ZIP and use Chinese copy only**

```tsx
<input
  type="file"
  className="hidden"
  ref={fileInputRef}
  onChange={handleFileChange}
  accept=".pdf,.docx,.md,.markdown,.txt,.zip"
  disabled={store.isScanning || store.isAnalyzing || store.isGenerating}
/>

<p className="text-sm text-zinc-500 dark:text-zinc-400 text-center mb-2">
  拖拽 PDF、DOCX、Markdown、TXT 或 ZIP 课件包到这里
</p>
```

- [ ] **Step 5: Run the frontend test to verify it passes**

Run: `cd frontend && npm test -- src/hooks/generator/fileParserUtils.test.ts`  
Expected: PASS for both `source_content` and `archive_summary` extraction.

- [ ] **Step 6: Commit the frontend slice**

```bash
git add frontend/src/components/generator/GeneratorInput.tsx \
  frontend/src/hooks/generator/fileParserUtils.ts \
  frontend/src/hooks/generator/fileParserUtils.test.ts \
  frontend/src/hooks/generator/useFileParser.ts
git commit -m "feat(frontend): support zip courseware uploads"
```

### Task 4: Sync documentation and run end-to-end verification

**Files:**
- Modify: `.trae/documents/InkWords_API.md`
- Modify: `.trae/documents/InkWords_Architecture.md`
- Modify: `.trae/documents/InkWords_Conversation_Log.md`
- Modify: `.trae/documents/InkWords_Development_Plan_and_Log.md`
- Modify: `.trae/documents/InkWords_PRD.md`
- Modify: `README.md`

- [ ] **Step 1: Update API and architecture docs with ZIP parse behavior**

```md
### POST /api/v1/project/parse
- 新增支持：`.zip` 课件包
- 返回：
  - `data.source_content`
  - `data.archive_summary`（仅 ZIP 场景出现）
- 失败条件：
  - 压缩包中没有可解析的文本文件
  - 压缩包存在非法路径条目
```

- [ ] **Step 2: Update product and development logs with the new feature**

```md
- 2026-05-21：新增 ZIP 课件包解析设计与实现计划，支持文档/代码文本自动去重聚合并输出解析摘要。
```

- [ ] **Step 3: Run targeted tests and the containerized verification flow**

Run: `cd backend && go test ./internal/infra/parser ./internal/domain/project ./internal/domain/stream -v && cd ../frontend && npm test -- src/hooks/generator/fileParserUtils.test.ts`  
Expected: PASS for backend and frontend focused tests.

Run: `docker compose down && docker compose up -d --build`  
Expected: containers restart successfully and the app is reachable at `http://localhost`.

- [ ] **Step 4: Manually verify the user flow**

Run through this checklist:

```text
1. 登录系统并进入生成页。
2. 上传一个包含 md/txt/go/重复 txt 的 ZIP。
3. 观察“正在上传并解析文件...”后出现压缩包摘要提示。
4. 确认系统继续进入大纲生成，不再误判为 git 来源。
5. 再上传一个普通 PDF，确认行为与现有版本一致，且没有 archive_summary 提示。
```

- [ ] **Step 5: Review staged docs diff and create the final commit**

Run: `git diff --staged`  
Expected: only ZIP courseware parsing code, tests, and doc updates are present.

```bash
git add .trae/documents/InkWords_API.md \
  .trae/documents/InkWords_Architecture.md \
  .trae/documents/InkWords_Conversation_Log.md \
  .trae/documents/InkWords_Development_Plan_and_Log.md \
  .trae/documents/InkWords_PRD.md \
  README.md
git commit -m "docs: document zip courseware parsing flow"
```

## Self-Review
- Spec coverage: 已覆盖 ZIP 上传、白名单解析、去重、摘要返回、前端提示、安全限制、测试和文档同步，没有遗漏设计稿中的主要求。
- Placeholder scan: 计划中未使用 `TODO`、`TBD`、`similar to` 等占位词，每个任务都给出了明确文件、命令和代码片段。
- Type consistency:
  - 后端统一使用 `ParseResult`、`ParsedSource`、`ArchiveSummary`。
  - 前端统一使用 `archive_summary` 字段名，与后端 JSON 保持一致。
