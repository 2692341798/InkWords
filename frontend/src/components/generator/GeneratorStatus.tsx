import { useRef, useEffect } from 'react'
import { MarkdownEngine } from '@/components/MarkdownEngine'
import { useStreamStore } from '@/store/streamStore'
import { useShallow } from 'zustand/react/shallow'

import { CheckCircle2, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'

export function GeneratorStatus() {
  const store = useStreamStore(
    useShallow((state) => ({
      isScanning: state.isScanning,
      isAnalyzing: state.isAnalyzing,
      isGenerating: state.isGenerating,
      progress: state.progress,
      currentChapterTitle: state.currentChapterTitle,
      analysisMessage: state.analysisMessage,
      analysisHistory: state.analysisHistory,
      analysisStep: state.analysisStep,
      sourceType: state.sourceType,
      outline: state.outline,
      content: state.content,
      chapterStatus: state.chapterStatus,
      chapterContents: state.chapterContents,
      chapterErrors: state.chapterErrors,
    })),
  )
  const contentEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (contentEndRef.current) {
      contentEndRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [store.progress, store.currentChapterTitle, store.analysisHistory.length])

  if (!store.isScanning && !store.isAnalyzing && !store.isGenerating && !store.progress && !store.currentChapterTitle && !store.analysisMessage) {
    return null
  }

  const isParsing = store.isAnalyzing || store.isScanning
  const isWorking = isParsing || store.isGenerating
  const title = isParsing ? '解析进度' : '生成进度'
  const currentStatusMessage = store.isScanning ? store.analysisMessage : (isParsing ? store.analysisMessage : store.progress)

  // Also hide if there is no parsing/generating progress and no content to show
  if (!isWorking && !store.content && !currentStatusMessage) {
    return null
  }

  // Why: Task 3 turns progress into embeddable page content, so this component
  // should only own the progress card itself and not a fullscreen overlay shell.
  return (
    <div className="overflow-hidden rounded-xl border border-border bg-card">
      <div className="flex items-center justify-between border-b border-border bg-secondary/35 px-5 py-4">
        <h3 className="flex items-center gap-2 font-medium text-foreground">
          <span className={`h-2.5 w-2.5 rounded-full ${isWorking ? 'bg-[var(--brand)]' : 'bg-[var(--success)]'}`}></span>
          {title}
        </h3>
        {store.currentChapterTitle && !isParsing && (
          <span className="status-pill">
            正在生成: {store.currentChapterTitle}
          </span>
        )}
      </div>
      <div className="max-h-[70vh] overflow-y-auto p-5 custom-scrollbar">
        <div className="prose max-w-none bg-transparent dark:prose-invert">
        {!isParsing && currentStatusMessage && (
          <div className="mb-5 flex items-center gap-3 rounded-xl border border-border bg-secondary/35 p-4 font-mono text-sm text-foreground">
            {isWorking && (
              <Loader2 className="w-5 h-5 animate-spin flex-shrink-0" />
            )}
            <span className="break-all">{currentStatusMessage}</span>
          </div>
        )}

        {isParsing ? (
          <div className="flex flex-col gap-4">
             {store.analysisStep >= 0 && (
               <div className="w-full rounded-xl border border-border bg-card p-5">
                 <div className="flex justify-between text-sm font-medium mb-3 text-foreground">
                   <span className={cn("transition-colors", store.analysisStep >= 0 ? "text-[var(--brand)]" : "text-muted-foreground")}>{store.sourceType === 'file' ? '上传文件' : '克隆代码'}</span>
                   <span className={cn("transition-colors", store.analysisStep >= 1 ? "text-[var(--brand)]" : "text-muted-foreground")}>{store.sourceType === 'file' ? '解析内容' : '扫描结构'}</span>
                   <span className={cn("transition-colors", store.analysisStep >= 2 ? "text-[var(--brand)]" : "text-muted-foreground")}>深度分析</span>
                   <span className={cn("transition-colors", store.analysisStep >= 3 ? "text-[var(--brand)]" : "text-muted-foreground")}>生成大纲</span>
                 </div>
                 <div className="h-2 w-full overflow-hidden rounded-full bg-secondary">
                  <div 
                    className="h-full rounded-full bg-[var(--brand)] transition-all duration-500"
                    style={{ width: `${Math.min(100, Math.max(5, (store.analysisStep + 1) * 25))}%` }}
                  />
                </div>
              </div>
            )}
            
            <div className="space-y-3">
              {store.analysisHistory.map((item, index) => {
                const isLast = index === store.analysisHistory.length - 1;
                const isCompleted = !isLast || store.analysisStep >= 4;
                return (
                  <div 
                    key={item.id} 
                    className={cn(
                      "flex items-start gap-4 rounded-xl border p-4 transition-colors duration-200",
                      isCompleted
                        ? "border-border bg-card opacity-80"
                        : "border-[color-mix(in_srgb,var(--brand)_24%,var(--border))] bg-[var(--brand-soft)]"
                    )}
                  >
                    <div className="mt-0.5 flex-shrink-0">
                      {isCompleted ? (
                        <CheckCircle2 className="w-5 h-5 text-[var(--success)]" />
                      ) : (
                        <Loader2 className="w-5 h-5 text-[var(--brand)] animate-spin" />
                      )}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className={cn(
                        "text-sm font-medium break-all",
                        isCompleted ? "text-muted-foreground" : "text-foreground"
                      )}>
                        {item.message}
                      </div>
                      {item.status && (
                        <div className="text-xs text-zinc-500 dark:text-zinc-400 mt-1.5 flex items-center gap-2">
                          <span className="rounded-md bg-secondary px-2 py-0.5 capitalize">{item.status}</span>
                        </div>
                      )}
                    </div>
                  </div>
                )
              })}
              <div ref={contentEndRef} />
            </div>
          </div>
        ) : (
          <div className="flex flex-col gap-4">
            {store.outline && store.outline.length > 0 ? (
              <div className="space-y-3 pb-8">
                {[...store.outline].sort((a,b)=>a.sort-b.sort).map(chapter => {
                  const status = store.chapterStatus[chapter.sort] || 'pending';
                  const content = store.chapterContents[chapter.sort] || '';
                  const errorMessage = store.chapterErrors[chapter.sort] || '';
                  const snippet = content.length > 80 ? '...' + content.slice(-80) : content;
                  return (
                    <div key={chapter.sort} className={cn(
                      "flex flex-col gap-3 rounded-xl border p-4 transition-colors duration-200",
                      status === 'completed' ? "border-border bg-card opacity-80" :
                      status === 'generating' ? "border-[color-mix(in_srgb,var(--brand)_24%,var(--border))] bg-[var(--brand-soft)]" :
                      "border-border bg-secondary/35 opacity-70"
                    )}>
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          {status === 'generating' ? <Loader2 className="w-5 h-5 text-[var(--brand)] animate-spin" /> :
                           status === 'completed' ? <CheckCircle2 className="w-5 h-5 text-[var(--success)]" /> :
                           <div className="w-5 h-5 rounded-full border-2 border-border" />}
                          <span className={cn(
                            "font-medium",
                            status === 'generating' ? "text-foreground" :
                            status === 'completed' ? "text-muted-foreground" :
                            "text-muted-foreground"
                          )}>{chapter.title}</span>
                        </div>
                        <span className="rounded-md bg-secondary px-2 py-1 text-xs capitalize text-muted-foreground">
                          {status === 'generating' ? '生成中' : status === 'completed' ? '已完成' : status === 'pending' ? '等待中' : status === 'error' ? '失败' : status}
                        </span>
                      </div>
                      {status === 'error' && errorMessage && (
                        <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300">
                          失败原因：{errorMessage}
                        </div>
                      )}
                      {snippet && (
                        <div className="line-clamp-2 rounded-lg border border-border bg-secondary/35 p-3 font-mono text-sm leading-relaxed text-muted-foreground">
                          {snippet}
                          {status === 'generating' && <span className="ml-1 inline-block h-3 w-1.5 align-middle bg-[var(--brand)]" />}
                        </div>
                      )}
                    </div>
                  )
                })}
                {/* Render series intro chapter if it exists */}
                {store.chapterStatus[0] && (
                  <div className={cn(
                    "flex flex-col gap-3 rounded-xl border p-4 transition-colors duration-200",
                    store.chapterStatus[0] === 'completed' ? "border-border bg-card opacity-80" :
                    store.chapterStatus[0] === 'generating' ? "border-[color-mix(in_srgb,var(--brand)_24%,var(--border))] bg-[var(--brand-soft)]" :
                    "border-border bg-secondary/35 opacity-70"
                  )}>
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        {store.chapterStatus[0] === 'generating' ? <Loader2 className="w-5 h-5 text-[var(--brand)] animate-spin" /> :
                         store.chapterStatus[0] === 'completed' ? <CheckCircle2 className="w-5 h-5 text-[var(--success)]" /> :
                         <div className="w-5 h-5 rounded-full border-2 border-border" />}
                        <span className={cn(
                          "font-medium",
                          store.chapterStatus[0] === 'generating' ? "text-foreground" :
                          store.chapterStatus[0] === 'completed' ? "text-muted-foreground" :
                          "text-muted-foreground"
                        )}>系列导读</span>
                      </div>
                      <span className="rounded-md bg-secondary px-2 py-1 text-xs capitalize text-muted-foreground">
                        {store.chapterStatus[0] === 'generating' ? '生成中' : store.chapterStatus[0] === 'completed' ? '已完成' : store.chapterStatus[0] === 'pending' ? '等待中' : store.chapterStatus[0] === 'error' ? '失败' : store.chapterStatus[0]}
                      </span>
                    </div>
                    {store.chapterStatus[0] === 'error' && store.chapterErrors[0] && (
                      <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300">
                        失败原因：{store.chapterErrors[0]}
                      </div>
                    )}
                    {store.chapterContents[0] && (
                      <div className="line-clamp-2 rounded-lg border border-border bg-secondary/35 p-3 font-mono text-sm leading-relaxed text-muted-foreground">
                        {store.chapterContents[0].length > 80 ? '...' + store.chapterContents[0].slice(-80) : store.chapterContents[0]}
                        {store.chapterStatus[0] === 'generating' && <span className="ml-1 inline-block h-3 w-1.5 align-middle bg-[var(--brand)]" />}
                      </div>
                    )}
                  </div>
                )}
                <div ref={contentEndRef} />
              </div>
            ) : (
              <div className="min-h-[400px] rounded-xl border border-border bg-card p-6">
                {store.content ? (
                  <MarkdownEngine content={store.content} />
                ) : (
                  <div className="flex h-full items-center justify-center py-12 text-sm text-muted-foreground">
                    {isWorking ? '等待大模型响应中...' : '暂无生成内容'}
                  </div>
                )}
                <div ref={contentEndRef} />
              </div>
            )}
          </div>
        )}
        </div>
      </div>
    </div>
  )
}
