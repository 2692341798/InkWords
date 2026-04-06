# Blog Enhancements Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enhance concurrency visualization with worker slots, make the blog outline editable before generation, and adjust LLM prompts for unlimited, highly-detailed technical posts.

**Architecture:** We will modify the Go backend to assign specific `worker_id`s to the Map-Reduce goroutines and update the LLM prompts. In the React frontend, we will update the Zustand store to handle an array of active workers and editable chapter actions, and update the Generator UI to display the new worker slots and inline-editable outline cards.

**Tech Stack:** Go 1.24, Gin, React 18, Zustand, Tailwind CSS, Shadcn UI

---

### Task 1: Backend - Concurrency Worker IDs & Prompts

**Files:**
- Modify: `backend/internal/service/decomposition.go`
- Modify: `backend/internal/service/generator.go`

- [ ] **Step 1: Update Outline Generation Prompt**

Modify `GenerateOutline` in `backend/internal/service/decomposition.go`:
```go
// Replace the old prompt with the new one
prompt := fmt.Sprintf(`你是一个高级架构师。请评估以下项目文本，并生成一个系列博客的大纲。
对于大型项目、源码仓库或复杂内容，**强制拆分为细粒度系列博客**。
要求一个技术点分为一个博客，博客篇数上不封顶，只要有需要，技术点可以拆的更加详细。
输出必须是纯JSON格式，包含 series_title 和 chapters 两个字段，不包含任何Markdown标记或其他文字。
JSON 格式如下：
{
  "series_title": "系列博客的标题",
  "chapters": [
    {
      "title": "章节标题",
      "summary": "该章节的详细摘要或内容要点（指导后续生成的具体方向）",
      "sort": 1,
      "files": ["强相关的具体文件路径或目录（必须是相对路径）"]
    }
  ]
}

项目文本：
%s`, sourceContent)
```

- [ ] **Step 2: Update Blog Generation Prompt**

Modify `GenerateBlogStream` in `backend/internal/service/generator.go`:
```go
prompt := fmt.Sprintf(`你是一个高级全栈架构师和技术博主。请根据以下提供的源内容，将其转化为一篇“小白友好、图文并茂、可独立复现”的高质量技术博客。
要求：
1. **字数充足，内容详实**：不要只写干瘪的总结。必须深入分析实现原理。
2. **代码级剖析**：对于每个技术点都添加更多的代码样例和图片来解释的更加详细。如果源内容包含代码，请引用核心代码并逐行解释其作用。
3. **可复现的步骤**：如果是实战或教程相关，请给出明确的执行步骤。
4. **小白友好**：在解释抽象的理论概念时，必须提供对应的代码示例或生活化比喻。
5. 所有生成的 Mermaid 图表代码块绝对禁止包含自定义样式关键字（如 style, classDef, linkStyle 等），必须使用基础语法。

源内容：
%s`, sourceContent)
```

- [ ] **Step 3: Add worker_id to Map-Reduce**

Modify `mapReduceAnalyze` in `backend/internal/service/decomposition.go`:
```go
func (s *DecompositionService) mapReduceAnalyze(ctx context.Context, chunks []parser.FileChunk, sendProgress func(int, string, interface{})) []string {
var summaries []string
var mu sync.Mutex

sem := semaphore.NewWeighted(5) // Max 5 concurrent goroutines
var wg sync.WaitGroup

workerPool := make(chan int, 5)
for i := 0; i < 5; i++ {
workerPool <- i
}

for i, chunk := range chunks {
wg.Add(1)
go func(idx int, c parser.FileChunk) {
defer wg.Done()
if err := sem.Acquire(ctx, 1); err != nil {
return
}
workerID := <-workerPool
defer func() {
workerPool <- workerID
sem.Release(1)
}()

sendProgress(2, fmt.Sprintf("正在分析分块 %d/%d [%s]...", idx+1, len(chunks), c.Dir), map[string]interface{}{
"status":    "chunk_analyzing",
"dir":       c.Dir,
"index":     idx + 1,
"total":     len(chunks),
"worker_id": workerID,
})

summary := s.generateLocalSummaryWithRetry(ctx, c, 3, sendProgress, idx+1, len(chunks), workerID)

if summary != "" {
mu.Lock()
summaries = append(summaries, summary)
mu.Unlock()
sendProgress(2, fmt.Sprintf("分块 %d/%d 分析完成", idx+1, len(chunks)), map[string]interface{}{
"status":    "chunk_done",
"dir":       c.Dir,
"index":     idx + 1,
"worker_id": workerID,
})
}
}(i, chunk)
}

wg.Wait()
return summaries
}
```
Update `generateLocalSummaryWithRetry` signature and body to pass `workerID` to `sendProgress` events:
```go
func (s *DecompositionService) generateLocalSummaryWithRetry(ctx context.Context, chunk parser.FileChunk, maxRetries int, sendProgress func(int, string, interface{}), idx int, total int, workerID int) string {
// ...
sendProgress(2, fmt.Sprintf("分块 %d/%d 分析失败，正在重试 (%d/%d)...", idx, total, attempt, maxRetries), map[string]interface{}{
"status":    "chunk_failed",
"dir":       chunk.Dir,
"index":     idx,
"attempt":   attempt,
"worker_id": workerID,
})
// ...
sendProgress(2, fmt.Sprintf("分块 %d/%d 分析最终失败，已跳过", idx, total), map[string]interface{}{
"status":    "chunk_failed_final",
"dir":       chunk.Dir,
"index":     idx,
"worker_id": workerID,
})
return ""
}
```

### Task 2: Frontend - Zustand Store Updates

**Files:**
- Modify: `frontend/src/store/streamStore.ts`

- [ ] **Step 1: Update MapReduceProgress Interface**

```typescript
export interface MapReduceProgress {
  status: 'chunk_analyzing' | 'chunk_done' | 'chunk_failed' | 'chunk_failed_final' | ''
  dir: string
  index: number
  total: number
  attempt?: number
  worker_id?: number
}
```

- [ ] **Step 2: Update Store State and Actions**

```typescript
interface StreamState {
  // ... existing state ...
  workers: Record<number, MapReduceProgress>;
  
  // ... existing actions ...
  setMapReduceProgress: (progress: MapReduceProgress) => void
  updateChapter: (sort: number, field: 'title' | 'summary', value: string) => void
  addChapter: () => void
  removeChapter: (sort: number) => void
  moveChapter: (sort: number, direction: 'up' | 'down') => void
}

export const useStreamStore = create<StreamState>()((set, get) => ({
  // ... existing state ...
  workers: {},

  setMapReduceProgress: (progress) => set((state) => {
    if (progress.worker_id !== undefined) {
      return {
        mapReduceProgress: progress,
        workers: {
          ...state.workers,
          [progress.worker_id]: progress
        }
      }
    }
    return { mapReduceProgress: progress }
  }),

  updateChapter: (sort, field, value) => set((state) => ({
    outline: state.outline?.map(ch => 
      ch.sort === sort ? { ...ch, [field]: value } : ch
    )
  })),

  addChapter: () => set((state) => {
    if (!state.outline) return state;
    const maxSort = state.outline.reduce((max, ch) => Math.max(max, ch.sort), 0);
    const newChapter: Chapter = {
      sort: maxSort + 1,
      title: '新章节标题',
      summary: '请填写章节摘要...',
      files: []
    };
    return { outline: [...state.outline, newChapter] };
  }),

  removeChapter: (sort) => set((state) => {
    if (!state.outline) return state;
    const newOutline = state.outline
      .filter(ch => ch.sort !== sort)
      .map((ch, index) => ({ ...ch, sort: index + 1 })); // Re-sort
    return { outline: newOutline };
  }),

  moveChapter: (sort, direction) => set((state) => {
    if (!state.outline) return state;
    const index = state.outline.findIndex(ch => ch.sort === sort);
    if (
      (direction === 'up' && index === 0) || 
      (direction === 'down' && index === state.outline.length - 1)
    ) return state;

    const newOutline = [...state.outline];
    const swapIndex = direction === 'up' ? index - 1 : index + 1;
    
    // Swap elements
    [newOutline[index], newOutline[swapIndex]] = [newOutline[swapIndex], newOutline[index]];
    
    // Re-sort
    const sortedOutline = newOutline.map((ch, i) => ({ ...ch, sort: i + 1 }));
    return { outline: sortedOutline };
  }),
  
  reset: () => set({
    // ... existing reset state ...
    workers: {},
  }),
}))
```

### Task 3: Frontend - Generator UI Updates

**Files:**
- Modify: `frontend/src/components/Generator.tsx`

- [ ] **Step 1: Implement Worker Slots UI**

Replace the single progress block with 5 worker slots:
```tsx
import { Loader2, CheckCircle2, XCircle, AlertCircle, ArrowUp, ArrowDown, Trash2, Plus } from 'lucide-react'

// Inside Generator component:
{store.analysisStep === 2 && Object.keys(store.workers).length > 0 && (
  <div className="mt-4 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3">
    {[0, 1, 2, 3, 4].map(workerId => {
      const worker = store.workers[workerId];
      if (!worker) return (
        <div key={workerId} className="h-20 bg-zinc-50 border border-zinc-100 rounded-lg flex items-center justify-center text-zinc-300 text-xs">
          Worker {workerId + 1} 闲置
        </div>
      );
      
      const isAnalyzing = worker.status === 'chunk_analyzing';
      const isFailed = worker.status === 'chunk_failed';
      const isDone = worker.status === 'chunk_done';
      
      return (
        <div key={workerId} className={`p-3 rounded-lg border text-sm transition-all duration-300 ${
          isAnalyzing ? 'bg-indigo-50 border-indigo-200 shadow-[0_0_10px_rgba(99,102,241,0.2)] animate-pulse' :
          isFailed ? 'bg-orange-50 border-orange-200' :
          isDone ? 'bg-green-50 border-green-200' : 'bg-zinc-50 border-zinc-200'
        }`}>
          <div className="flex justify-between items-center mb-1">
            <span className="text-xs font-medium text-zinc-500">Worker {workerId + 1}</span>
            <span className={
              isFailed ? 'text-orange-500' : 
              worker.status === 'chunk_failed_final' ? 'text-red-500' : 
              isDone ? 'text-green-500' : 'text-indigo-500 font-medium'
            }>
              {isFailed ? `重试 (${worker.attempt}/3)` :
               worker.status === 'chunk_failed_final' ? '跳过' :
               isDone ? '完成' : '分析中'}
            </span>
          </div>
          <div className="truncate font-mono text-xs text-zinc-600" title={worker.dir}>
            {worker.dir}
          </div>
          <div className="text-xs text-zinc-400 mt-1">
            {worker.index} / {worker.total}
          </div>
        </div>
      );
    })}
  </div>
)}
```

- [ ] **Step 2: Implement Editable Outline UI**

Replace the static outline map `store.outline.map((ch) => ...)` with:
```tsx
<div className="space-y-4 max-h-[50vh] overflow-y-auto custom-scrollbar pr-2">
  {store.outline.map((ch, index) => (
    <div key={ch.sort} className="p-4 bg-white rounded-xl border border-zinc-200 shadow-sm hover:border-indigo-200 transition-colors">
      <div className="flex items-start gap-3 mb-3">
        <div className="w-6 h-6 rounded-full bg-indigo-100 text-indigo-700 flex items-center justify-center text-sm font-semibold shrink-0 mt-1">
          {ch.sort}
        </div>
        <div className="flex-1 min-w-0">
          <input
            type="text"
            value={ch.title}
            onChange={(e) => store.updateChapter(ch.sort, 'title', e.target.value)}
            className="w-full font-medium text-zinc-900 border-none bg-transparent focus:outline-none focus:ring-0 p-0 text-base"
            placeholder="章节标题"
            disabled={store.isGenerating}
          />
        </div>
        <div className="flex items-center gap-1 shrink-0 opacity-0 hover:opacity-100 focus-within:opacity-100 group-hover:opacity-100 transition-opacity">
          <button 
            onClick={() => store.moveChapter(ch.sort, 'up')}
            disabled={index === 0 || store.isGenerating}
            className="p-1.5 text-zinc-400 hover:text-indigo-600 hover:bg-indigo-50 rounded disabled:opacity-30"
          >
            <ArrowUp className="w-4 h-4" />
          </button>
          <button 
            onClick={() => store.moveChapter(ch.sort, 'down')}
            disabled={index === store.outline!.length - 1 || store.isGenerating}
            className="p-1.5 text-zinc-400 hover:text-indigo-600 hover:bg-indigo-50 rounded disabled:opacity-30"
          >
            <ArrowDown className="w-4 h-4" />
          </button>
          <button 
            onClick={() => store.removeChapter(ch.sort)}
            disabled={store.outline!.length <= 1 || store.isGenerating}
            className="p-1.5 text-zinc-400 hover:text-red-600 hover:bg-red-50 rounded disabled:opacity-30"
          >
            <Trash2 className="w-4 h-4" />
          </button>
        </div>
      </div>
      <div className="pl-9">
        <textarea
          value={ch.summary}
          onChange={(e) => store.updateChapter(ch.sort, 'summary', e.target.value)}
          className="w-full text-sm text-zinc-600 bg-zinc-50 border border-transparent hover:border-zinc-200 focus:border-indigo-300 focus:bg-white rounded-md p-2 resize-y min-h-[60px] focus:outline-none transition-colors"
          placeholder="章节内容摘要或要点..."
          disabled={store.isGenerating}
        />
      </div>
    </div>
  ))}
  <button
    onClick={() => store.addChapter()}
    disabled={store.isGenerating}
    className="w-full py-3 border-2 border-dashed border-zinc-200 rounded-xl text-zinc-500 hover:text-indigo-600 hover:border-indigo-300 hover:bg-indigo-50/50 transition-all flex items-center justify-center gap-2 font-medium"
  >
    <Plus className="w-4 h-4" />
    添加新章节
  </button>
</div>
```
Ensure icons are imported from `lucide-react` at the top of the file.

