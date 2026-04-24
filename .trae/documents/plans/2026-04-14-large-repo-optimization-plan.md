# 大型仓库解析优化与优雅降级方案 (Large Repo Module Parsing) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Provide accurate subdirectory fetching (sparse checkout) for large git repositories and graceful degradation based on chunk count, coupled with concurrency enhancements to avoid LLM API rate limits.

**Architecture:**
1. **API/Models:** Add `sub_dir` to `AnalyzeRequest` and `GenerateRequest`.
2. **GitFetcher:** Implement `git clone --filter=blob:none --no-checkout` followed by `git sparse-checkout set <sub_dir>` and `git checkout`. Apply a max chunk threshold of 15; if exceeded, inject a system prompt string indicating truncation.
3. **Concurrency:** Move the CPU-bound hardcoded worker limit to `MAX_CONCURRENT_WORKERS` in `.env`. Introduce `golang.org/x/time/rate` token bucket rate limiter based on `LLM_API_RPM_LIMIT` to throttle requests to the LLM. Implement concurrent tree reduction for intermediate summaries.
4. **Frontend:** Add an "Advanced Options" accordion in the UI to accept `sub_dir`.

**Tech Stack:** Go (Gin, `golang.org/x/time/rate`, `x/sync/semaphore`), React (Tailwind CSS, Shadcn UI Accordion, Zustand).

---

### Task 1: API Model & Environment Variables Update

**Files:**
- Modify: `backend/internal/api/project.go`
- Modify: `backend/internal/api/stream.go`

- [ ] **Step 1: Update API Request Models**
In `backend/internal/api/project.go`, update `AnalyzeRequest`:
```go
type AnalyzeRequest struct {
	GitURL string `json:"git_url" binding:"required"`
	SubDir string `json:"sub_dir"`
}
```

In `backend/internal/api/stream.go`, update `GenerateRequest`:
```go
type GenerateRequest struct {
	GitURL string `json:"git_url" binding:"required"`
	SubDir string `json:"sub_dir"`
}
```

- [ ] **Step 2: Pass SubDir to Service Methods**
In `backend/internal/api/project.go`, update the `Fetch` call:
```go
	treeContent, chunks, err := api.gitFetcher.Fetch(req.GitURL, req.SubDir)
```

In `backend/internal/api/stream.go`, pass `req.SubDir` to `AnalyzeStream`:
```go
	go func() {
		defer wg.Done()
		api.decompositionService.AnalyzeStream(bgCtx, req.GitURL, req.SubDir, progressChan, errChan)
	}()
```

### Task 2: GitFetcher Sparse Checkout & Truncation

**Files:**
- Modify: `backend/internal/parser/git_fetcher.go`
- Modify: `backend/internal/parser/git_fetcher_test.go`

- [ ] **Step 1: Update Fetch Signature and Implement Sparse Checkout**
Update `Fetch` signature:
```go
func (f *GitFetcher) Fetch(repoURL string, subDir string) (string, []FileChunk, error) {
```

Replace the `git clone --depth 1` command block with Sparse Checkout logic if `subDir` is provided:
```go
	if subDir != "" {
		// Sparse checkout
		cmd := exec.Command("git", "clone", "--filter=blob:none", "--no-checkout", repoURL, tempDir)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return "", nil, fmt.Errorf("failed to clone repository: %w, stderr: %s", err, stderr.String())
		}
		
		cmd = exec.Command("git", "sparse-checkout", "set", subDir)
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			return "", nil, fmt.Errorf("failed to set sparse-checkout: %w", err)
		}
		
		cmd = exec.Command("git", "checkout")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			return "", nil, fmt.Errorf("failed to checkout: %w", err)
		}
	} else {
		// Shallow clone
		cmd := exec.Command("git", "clone", "--depth", "1", repoURL, tempDir)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return "", nil, fmt.Errorf("failed to clone repository: %w, stderr: %s", err, stderr.String())
		}
	}
```

- [ ] **Step 2: Add SubDir Targeting & Truncation in filepath.Walk**
Update `filepath.Walk` start directory and track chunks:
```go
	walkDir := tempDir
	if subDir != "" {
		walkDir = filepath.Join(tempDir, subDir)
		if _, err := os.Stat(walkDir); os.IsNotExist(err) {
			return "", nil, fmt.Errorf("指定的子目录不存在: %s", subDir)
		}
	}

	maxTotalChunks := 15
	currentChunkCount := 0
	truncated := false

	err = filepath.Walk(walkDir, func(path string, info os.FileInfo, err error) error {
        if truncated {
            return filepath.SkipDir
        }
        // ... existing ignore logic ...
        // Before creating or appending chunk, check chunk count
        // For simplicity, we track `len(dirContents)` as rough chunk count, or track after loop
```
Wait, we should track it correctly. Modify the chunk splitting logic at the end of `GitFetcher.Fetch` instead to just slice the `chunks` array:
```go
	if len(chunks) > 15 {
		chunks = chunks[:15]
		treeBuilder.WriteString("\n\n【系统提示】由于该项目体量极其庞大，系统已执行优雅降级，自动截断了后续文件（仅保留了前15个核心模块的分块）。请你在生成的博客引言或开头中，自然地向读者说明：由于项目过于庞大，本文仅抽取分析了其核心的若干模块代码，并未包含全量内容。")
	}
```
*Note: Truncating the array is simpler and safer than complex early-exit in `filepath.Walk` since we need to guarantee exact chunk limits. To prevent OOM during Walk, you can still add an early exit in Walk if `len(dirContents) > 30`.*

- [ ] **Step 3: Update Tests**
In `git_fetcher_test.go`, update `Fetch` calls to pass `""` for `subDir`.

### Task 3: Concurrency Optimizations in Decomposition Service

**Files:**
- Modify: `backend/internal/service/decomposition.go`

- [ ] **Step 1: Update AnalyzeStream Signature**
Update `AnalyzeStream` signature:
```go
func (s *DecompositionService) AnalyzeStream(ctx context.Context, gitURL string, subDir string, progressChan chan<- string, errChan chan<- error) {
```
Pass `subDir` to `Fetch(gitURL, subDir)`.

- [ ] **Step 2: Implement Rate Limiter (Token Bucket)**
In `DecompositionService` struct, add `limiter *rate.Limiter`.
In `NewDecompositionService`:
```go
import "golang.org/x/time/rate"

// ...
rpm := 60
if rpmStr := os.Getenv("LLM_API_RPM_LIMIT"); rpmStr != "" {
    if val, err := strconv.Atoi(rpmStr); err == nil && val > 0 {
        rpm = val
    }
}
// rate.Limit is events per second.
limit := rate.Limit(float64(rpm) / 60.0)
limiter := rate.NewLimiter(limit, 1) // burst of 1

return &DecompositionService{
    llmClient:  llm.NewDeepSeekClient(apiKey),
    gitFetcher: parser.NewGitFetcher(),
    limiter:    limiter,
}
```

- [ ] **Step 3: Apply Rate Limiter**
In `generateLocalSummaryWithRetry` and anywhere `s.llmClient.Generate` is called, wrap it with `s.limiter.Wait(ctx)`:
```go
	if err := s.limiter.Wait(ctx); err != nil {
		return ""
	}
	summary, err := s.llmClient.Generate(attemptCtx, modelStr, messages)
```

- [ ] **Step 4: MAX_CONCURRENT_WORKERS from Environment**
In `mapReduceAnalyze`:
```go
	maxWorkers := 3
	if envWorkers := os.Getenv("MAX_CONCURRENT_WORKERS"); envWorkers != "" {
		if val, err := strconv.Atoi(envWorkers); err == nil && val > 0 {
			maxWorkers = val
		}
	}
	if len(chunks) < maxWorkers {
		maxWorkers = len(chunks)
	}
```

- [ ] **Step 5: Concurrent Tree Reduce**
In `AnalyzeStream`, where `Tree Reduce` happens:
```go
		if len(summaries) > 20 {
			sendProgress(2, fmt.Sprintf("局部摘要数量较多 (%d)，正在进行中间层 Tree Reduce 汇总...", len(summaries)), nil)
			
			batchSize := 10
			numBatches := (len(summaries) + batchSize - 1) / batchSize
			intermediateSummaries := make([]string, numBatches)
			
			// Use the same semaphore or a new one to limit concurrency
			treeSem := semaphore.NewWeighted(int64(maxWorkers))
			var treeWg sync.WaitGroup
			
			for i := 0; i < len(summaries); i += batchSize {
				treeWg.Add(1)
				go func(startIdx, batchIdx int) {
					defer treeWg.Done()
					if err := treeSem.Acquire(ctx, 1); err != nil {
						return
					}
					defer treeSem.Release(1)
					
					end := startIdx + batchSize
					if end > len(summaries) {
						end = len(summaries)
					}
					batchContent := strings.Join(summaries[startIdx:end], "\n\n")
					
					// ... Prepare prompt ...
					if err := s.limiter.Wait(ctx); err == nil {
					    ctxTimeout, cancel := context.WithTimeout(ctx, 3*time.Minute)
					    interSummary, err := s.llmClient.Generate(ctxTimeout, modelStr, req)
					    cancel()
					    if err != nil {
					        intermediateSummaries[batchIdx] = batchContent
					    } else {
					        intermediateSummaries[batchIdx] = interSummary
					    }
					}
				}(i, i/batchSize)
			}
			treeWg.Wait()
			
			// Flatten intermediateSummaries
			summaries = nil
			for _, s := range intermediateSummaries {
				if s != "" {
					summaries = append(summaries, s)
				}
			}
		}
```

### Task 4: Frontend UI Updates

**Files:**
- Modify: `frontend/src/hooks/useBlogStream.ts`
- Modify: `frontend/src/components/Generator.tsx`

- [ ] **Step 1: Update useBlogStream**
Update signature to `analyzeGit(gitUrl: string, subDir?: string)` and pass it in body:
```typescript
	body: JSON.stringify({ git_url: gitUrl, sub_dir: subDir || "" }),
```

- [ ] **Step 2: Add SubDir State and Accordion in Generator.tsx**
Import Shadcn UI Accordion components if they exist, or build a simple toggle UI if Shadcn Accordion is not initialized:
```tsx
import { ChevronDown, ChevronUp } from 'lucide-react'

// Inside component state
const [subDir, setSubDir] = useState('')
const [showAdvanced, setShowAdvanced] = useState(false)

// Update input call
const handleStartAnalysis = () => {
    // ...
    analyzeGit(gitUrl, subDir)
}

// Below the Git URL input
<div className="w-full mt-4">
  <button 
    onClick={() => setShowAdvanced(!showAdvanced)} 
    className="flex items-center text-sm text-zinc-500 hover:text-zinc-700"
  >
    高级选项 {showAdvanced ? <ChevronUp className="w-4 h-4 ml-1" /> : <ChevronDown className="w-4 h-4 ml-1" />}
  </button>
  {showAdvanced && (
    <div className="mt-3 p-4 bg-zinc-50 rounded-lg border border-zinc-100">
      <label className="block text-sm font-medium text-zinc-700 mb-1">
        指定解析子目录 (可选)
      </label>
      <input
        type="text"
        placeholder="如：src/net/http"
        className="w-full p-2 text-sm border border-zinc-200 rounded-md focus:ring-2 focus:ring-indigo-500 focus:border-transparent outline-none"
        value={subDir}
        onChange={(e) => setSubDir(e.target.value)}
      />
      <p className="text-xs text-zinc-500 mt-2">对于大型开源仓库，建议指定具体模块路径以加速解析并避免超限。</p>
    </div>
  )}
</div>
```
