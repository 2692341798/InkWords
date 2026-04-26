import { useRef, useEffect } from 'react'
import { MarkdownEngine } from '@/components/MarkdownEngine'
import { useStreamStore } from '@/store/streamStore'

import { CheckCircle2, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'

export function GeneratorStatus() {
  const store = useStreamStore()
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

  // Only render GeneratorStatus if we are actually doing work or have content.
  // We avoid rendering the loading UI multiple times.
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-900/60 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="bg-white dark:bg-zinc-900 rounded-2xl shadow-2xl border border-zinc-200 dark:border-zinc-800 overflow-hidden flex flex-col w-full max-w-2xl h-[80vh] max-h-[600px] animate-in zoom-in-95 duration-200">
        <div className="p-4 border-b border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-800/50 flex items-center justify-between">
          <h3 className="font-medium text-zinc-900 dark:text-zinc-100 flex items-center gap-2">
            <span className={`w-2.5 h-2.5 rounded-full ${isWorking ? 'bg-blue-500 dark:bg-blue-400 animate-pulse' : 'bg-green-500 dark:bg-green-400'}`}></span>
            {title}
          </h3>
          {store.currentChapterTitle && !isParsing && (
            <span className="text-sm text-zinc-500 dark:text-zinc-400 bg-white dark:bg-zinc-800 px-3 py-1 rounded-full border border-zinc-200 dark:border-zinc-700 shadow-sm">
              正在生成: {store.currentChapterTitle}
            </span>
          )}
        </div>
        <div className="flex-1 overflow-y-auto p-6 bg-zinc-50/30 dark:bg-zinc-900/30 prose dark:prose-invert max-w-none custom-scrollbar">
        {!isParsing && currentStatusMessage && (
          <div className="mb-6 p-4 bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300 rounded-lg border border-blue-100 dark:border-blue-800/50 font-mono text-sm shadow-sm flex items-center gap-3">
            {isWorking && (
              <Loader2 className="w-5 h-5 animate-spin flex-shrink-0" />
            )}
            <span className="break-all">{currentStatusMessage}</span>
          </div>
        )}
        
        {isParsing ? (
          <div className="flex flex-col gap-4">
             {store.analysisStep >= 0 && (
               <div className="w-full bg-white dark:bg-zinc-900 p-6 rounded-xl border border-zinc-200 dark:border-zinc-800 shadow-sm">
                 <div className="flex justify-between text-sm font-medium mb-3">
                   <span className={cn("transition-colors", store.analysisStep >= 0 ? "text-blue-600 dark:text-blue-400" : "text-zinc-400")}>{store.sourceType === 'file' ? '上传文件' : '克隆代码'}</span>
                   <span className={cn("transition-colors", store.analysisStep >= 1 ? "text-blue-600 dark:text-blue-400" : "text-zinc-400")}>{store.sourceType === 'file' ? '解析内容' : '扫描结构'}</span>
                   <span className={cn("transition-colors", store.analysisStep >= 2 ? "text-blue-600 dark:text-blue-400" : "text-zinc-400")}>深度分析</span>
                   <span className={cn("transition-colors", store.analysisStep >= 3 ? "text-blue-600 dark:text-blue-400" : "text-zinc-400")}>生成大纲</span>
                 </div>
                 <div className="w-full bg-zinc-100 dark:bg-zinc-800 rounded-full h-2 overflow-hidden">
                  <div 
                    className="bg-blue-500 h-full rounded-full transition-all duration-500 relative overflow-hidden"
                    style={{ width: `${Math.min(100, Math.max(5, (store.analysisStep + 1) * 25))}%` }}
                  >
                    <div className="absolute inset-0 bg-white/20 animate-pulse -skew-x-12" style={{ backgroundImage: 'linear-gradient(90deg, transparent, rgba(255,255,255,0.4), transparent)' }}></div>
                  </div>
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
                      "p-4 rounded-xl border flex items-start gap-4 transition-all duration-300 shadow-sm",
                      isCompleted
                        ? "bg-white dark:bg-zinc-800 border-zinc-200 dark:border-zinc-700 opacity-80" 
                        : "bg-blue-50/50 dark:bg-blue-900/10 border-blue-200 dark:border-blue-800/50 scale-[1.02]"
                    )}
                  >
                    <div className="mt-0.5 flex-shrink-0">
                      {isCompleted ? (
                        <CheckCircle2 className="w-5 h-5 text-green-500 dark:text-green-400" />
                      ) : (
                        <Loader2 className="w-5 h-5 text-blue-500 dark:text-blue-400 animate-spin" />
                      )}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className={cn(
                        "text-sm font-medium break-all",
                        isCompleted ? "text-zinc-700 dark:text-zinc-300" : "text-blue-900 dark:text-blue-100"
                      )}>
                        {item.message}
                      </div>
                      {item.status && (
                        <div className="text-xs text-zinc-500 dark:text-zinc-400 mt-1.5 flex items-center gap-2">
                          <span className="px-2 py-0.5 bg-zinc-100 dark:bg-zinc-800 rounded-md capitalize">{item.status}</span>
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
          <div className="bg-white dark:bg-zinc-900 p-6 rounded-xl border border-zinc-200 dark:border-zinc-800 shadow-sm min-h-[400px]">
            {store.content ? (
              <MarkdownEngine content={store.content} />
            ) : (
              <div className="h-full flex items-center justify-center text-zinc-400 py-12">
                {isWorking ? '等待大模型响应中...' : '暂无生成内容'}
              </div>
            )}
            <div ref={contentEndRef} />
          </div>
        )}
      </div>
    </div>
    </div>
  )
}
