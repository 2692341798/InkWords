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

  if (!store.isScanning && !store.isAnalyzing && !store.isGenerating && !store.progress && !store.currentChapterTitle) {
    return null
  }

  return (
    <div className="bg-white dark:bg-zinc-900 rounded-xl shadow-sm border border-zinc-200 dark:border-zinc-800 overflow-hidden flex flex-col h-[600px]">
      <div className="p-4 border-b border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-800/50 flex items-center justify-between">
        <h3 className="font-medium text-zinc-900 dark:text-zinc-100 flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-blue-500 dark:bg-blue-400 animate-pulse"></span>
          生成进度
        </h3>
        {store.currentChapterTitle && (
          <span className="text-sm text-zinc-500 dark:text-zinc-400 bg-white dark:bg-zinc-800 px-3 py-1 rounded-full border border-zinc-200 dark:border-zinc-700 shadow-sm">
            正在生成: {store.currentChapterTitle}
          </span>
        )}
      </div>
      <div className="flex-1 overflow-y-auto p-6 bg-zinc-50/30 dark:bg-zinc-900/30 prose dark:prose-invert max-w-none">
        {store.progress && (
          <div className="mb-8 p-4 bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300 rounded-lg border border-blue-100 dark:border-blue-800/50 font-mono text-sm shadow-sm flex items-center gap-3">
            <svg className="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            {store.progress}
          </div>
        )}
        <div className="bg-white dark:bg-zinc-900 p-6 rounded-xl border border-zinc-200 dark:border-zinc-800 shadow-sm min-h-[400px]">
          <MarkdownEngine content={store.content} />
          <div ref={contentEndRef} />
        </div>
      </div>
    </div>
  )
}
