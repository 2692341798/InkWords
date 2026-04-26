import type { Dispatch, SetStateAction } from 'react'
import { Button } from '@/components/ui/button'
import { ArrowUp, ArrowDown, Trash2, Plus, ChevronDown, ChevronUp } from 'lucide-react'
import { useStreamStore } from '@/store/streamStore'

interface GeneratorOutlineProps {
  isOutlineExpanded: boolean
  setIsOutlineExpanded: Dispatch<SetStateAction<boolean>>
  setShowChapterDeleteConfirm: Dispatch<SetStateAction<number | null>>
  handleGenerate: () => void
  stopGenerating: () => void
}

export function GeneratorOutline({
  isOutlineExpanded,
  setIsOutlineExpanded,
  setShowChapterDeleteConfirm,
  handleGenerate,
  stopGenerating
}: GeneratorOutlineProps) {
  const store = useStreamStore()

  if (!store.outline || store.outline.length === 0) return null

  return (
    <div className="mb-12 bg-white dark:bg-zinc-900 rounded-xl shadow-sm border border-zinc-200 dark:border-zinc-800 overflow-hidden transition-all duration-300">
      <div 
        className="p-6 border-b border-zinc-200 dark:border-zinc-800 flex items-center justify-between cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/50 transition-colors"
        onClick={() => setIsOutlineExpanded(!isOutlineExpanded)}
      >
        <div className="flex items-center gap-4">
          <div className="w-10 h-10 bg-indigo-50 dark:bg-indigo-900/20 text-indigo-600 dark:text-indigo-400 rounded-full flex items-center justify-center">
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h7" />
            </svg>
          </div>
          <div>
            <h3 className="text-lg font-medium text-zinc-900 dark:text-zinc-100 flex items-center gap-3">
              博客大纲 ({store.outline.length} 篇)
              {isOutlineExpanded ? (
                <ChevronUp className="w-5 h-5 text-zinc-400 dark:text-zinc-500" />
              ) : (
                <ChevronDown className="w-5 h-5 text-zinc-400 dark:text-zinc-500" />
              )}
            </h3>
            <p className="text-sm text-zinc-500 dark:text-zinc-400 mt-1">
              可拖拽调整顺序，或点击生成开始写作
            </p>
          </div>
        </div>
        
        <div className="flex items-center gap-3" onClick={e => e.stopPropagation()}>
          {store.isGenerating ? (
            <Button
              variant="destructive"
              onClick={stopGenerating}
            >
              停止生成
            </Button>
          ) : (
            <Button
              onClick={handleGenerate}
              className="bg-indigo-600 hover:bg-indigo-700 text-white dark:bg-indigo-500 dark:hover:bg-indigo-600"
            >
              开始生成系列博客
            </Button>
          )}
        </div>
      </div>

      {isOutlineExpanded && (
        <div className="p-6 bg-zinc-50/50 dark:bg-zinc-900/50">
          <div className="space-y-4">
            {store.outline.map((chapter, index) => (
              <div
                key={index}
                className="bg-white dark:bg-zinc-800 p-5 rounded-lg border border-zinc-200 dark:border-zinc-700 shadow-sm hover:shadow-md transition-shadow group relative"
              >
                <div className="flex items-start gap-4">
                  <div className="flex-shrink-0 w-8 h-8 bg-zinc-100 dark:bg-zinc-700 text-zinc-600 dark:text-zinc-300 rounded-full flex items-center justify-center font-medium">
                    {index + 1}
                  </div>
                  <div className="flex-1 min-w-0">
                    <input
                      type="text"
                      className="w-full bg-transparent text-base font-medium text-zinc-900 dark:text-zinc-100 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400 rounded px-2 py-1 -ml-2 mb-2"
                      value={chapter.title}
                      onChange={(e) => {
                        if (!store.outline) return;
                        const newOutline = [...store.outline]
                        newOutline[index].title = e.target.value
                        store.setOutline(newOutline)
                      }}
                      disabled={store.isGenerating}
                    />
                    <textarea
                      className="w-full bg-transparent text-sm text-zinc-600 dark:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400 rounded px-2 py-1 -ml-2 resize-none h-20"
                      value={chapter.summary}
                      onChange={(e) => {
                        if (!store.outline) return;
                        const newOutline = [...store.outline]
                        newOutline[index].summary = e.target.value
                        store.setOutline(newOutline)
                      }}
                      disabled={store.isGenerating}
                    />
                  </div>
                  <div className="flex-shrink-0 flex flex-col gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
                      disabled={index === 0 || store.isGenerating}
                      onClick={() => {
                        if (!store.outline) return;
                        const newOutline = [...store.outline]
                        const temp = newOutline[index - 1]
                        newOutline[index - 1] = newOutline[index]
                        newOutline[index] = temp
                        store.setOutline(newOutline)
                      }}
                    >
                      <ArrowUp className="w-4 h-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
                      disabled={index === (store.outline ? store.outline.length - 1 : 0) || store.isGenerating}
                      onClick={() => {
                        if (!store.outline) return;
                        const newOutline = [...store.outline]
                        const temp = newOutline[index + 1]
                        newOutline[index + 1] = newOutline[index]
                        newOutline[index] = temp
                        store.setOutline(newOutline)
                      }}
                    >
                      <ArrowDown className="w-4 h-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 text-red-500 hover:text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:text-red-300 dark:hover:bg-red-900/20"
                      disabled={store.isGenerating}
                      onClick={() => setShowChapterDeleteConfirm(index)}
                    >
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
                </div>
              </div>
            ))}
          </div>

          <div className="mt-6 flex justify-center">
            <Button
              variant="outline"
              className="text-zinc-600 dark:text-zinc-400 border-dashed hover:border-solid w-full max-w-md bg-white dark:bg-zinc-800 hover:bg-zinc-50 dark:hover:bg-zinc-700"
              disabled={store.isGenerating}
              onClick={() => {
                if (!store.outline) return;
                store.setOutline([
                  ...store.outline,
                  {
                    title: '新章节标题',
                    summary: '请在此输入章节概要，指导 AI 生成内容...',
                    sort: store.outline.length + 1,
                    files: []
                  }
                ])
              }}
            >
              <Plus className="w-4 h-4 mr-2" />
              添加新章节
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}