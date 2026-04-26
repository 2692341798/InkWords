import { useRef, useEffect } from 'react'
import { MarkdownEngine } from '@/components/MarkdownEngine'
import { useStreamStore } from '@/store/streamStore'

export function GeneratorStatus() {
  const store = useStreamStore()
  const contentEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (contentEndRef.current) {
      contentEndRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [store.progress, store.currentChapterTitle])

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
        {currentStatusMessage && (
          <div className="mb-6 p-4 bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300 rounded-lg border border-blue-100 dark:border-blue-800/50 font-mono text-sm shadow-sm flex items-center gap-3">
            {isWorking && (
              <svg className="w-5 h-5 animate-spin flex-shrink-0" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
              </svg>
            )}
            <span className="break-all">{currentStatusMessage}</span>
          </div>
        )}
        
        {isParsing ? (
          <div className="bg-white dark:bg-zinc-900 p-6 rounded-xl border border-zinc-200 dark:border-zinc-800 shadow-sm min-h-[400px] flex flex-col items-center justify-center text-zinc-500">
            <svg className="w-12 h-12 mb-4 text-blue-500 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <p className="text-sm">正在深度解析项目结构与内容，请耐心等待...</p>
             {store.analysisStep >= 0 && (
               <div className="mt-6 w-full max-w-md">
                 <div className="flex justify-between text-xs mb-2">
                   <span className={store.analysisStep >= 0 ? "text-blue-500" : ""}>{store.sourceType === 'file' ? '上传' : '克隆'}</span>
                   <span className={store.analysisStep >= 1 ? "text-blue-500" : ""}>{store.sourceType === 'file' ? '解析' : '扫描'}</span>
                   <span className={store.analysisStep >= 2 ? "text-blue-500" : ""}>分析</span>
                   <span className={store.analysisStep >= 3 ? "text-blue-500" : ""}>大纲</span>
                 </div>
                 <div className="w-full bg-zinc-200 dark:bg-zinc-700 rounded-full h-1.5">
                  <div 
                    className="bg-blue-500 h-1.5 rounded-full transition-all duration-500"
                    style={{ width: `${Math.min(100, Math.max(5, (store.analysisStep + 1) * 25))}%` }}
                  ></div>
                </div>
              </div>
            )}
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
