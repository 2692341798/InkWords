import type { Dispatch, SetStateAction } from 'react'
import { Button } from '@/components/ui/button'
import { ArrowUp, ArrowDown, Trash2, Plus, ChevronDown, ChevronUp } from 'lucide-react'
import { useStreamStore } from '@/store/streamStore'
import { useShallow } from 'zustand/react/shallow'

interface GeneratorOutlineProps {
  isOutlineExpanded: boolean
  setIsOutlineExpanded: Dispatch<SetStateAction<boolean>>
  setShowChapterDeleteConfirm: Dispatch<SetStateAction<number | null>>
  handleGenerate: () => void
  stopGenerating: () => void
  lockedScenarioLabel?: string | null
}

export function GeneratorOutline({
  isOutlineExpanded,
  setIsOutlineExpanded,
  setShowChapterDeleteConfirm,
  handleGenerate,
  stopGenerating,
}: GeneratorOutlineProps) {
  const store = useStreamStore(
    useShallow((state) => ({
      outline: state.outline,
      isGenerating: state.isGenerating,
      setOutline: state.setOutline,
    })),
  )

  if (!store.outline || store.outline.length === 0) return null

  return (
    <div className="overflow-hidden rounded-xl border border-border bg-card transition-colors">
      <div 
        className="flex cursor-pointer items-center justify-between border-b border-border px-5 py-4 transition-colors hover:bg-secondary/35"
        onClick={() => setIsOutlineExpanded(!isOutlineExpanded)}
      >
        <div className="flex items-center gap-4">
          <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-[var(--brand-soft)] text-[var(--brand)]">
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h7" />
            </svg>
          </div>
          <div>
            <h3 className="flex items-center gap-3 text-base font-semibold text-foreground">
              博客大纲 ({store.outline.length} 篇)
              {isOutlineExpanded ? (
                <ChevronUp className="w-5 h-5 text-muted-foreground" />
              ) : (
                <ChevronDown className="w-5 h-5 text-muted-foreground" />
              )}
            </h3>
            <p className="mt-1 text-sm text-muted-foreground">
              调整章节顺序和摘要，确认后开始生成
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
            >
              开始生成系列博客
            </Button>
          )}
        </div>
      </div>

      {isOutlineExpanded && (
        <div className="bg-secondary/25 p-5">
          <div className="space-y-3">
            {store.outline.map((chapter, index) => (
              <div
                key={index}
                className="group relative rounded-xl border border-border bg-card p-4 transition-colors hover:bg-secondary/20"
              >
                <div className="flex items-start gap-4">
                  <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg bg-secondary text-sm font-medium text-muted-foreground">
                    {index + 1}
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <input
                        type="text"
                        className="-ml-2 flex-1 rounded px-2 py-1 text-base font-medium text-foreground bg-transparent focus:outline-none focus:ring-2 focus:ring-ring"
                        value={chapter.title}
                        onChange={(e) => {
                          if (!store.outline) return;
                          const newOutline = [...store.outline]
                          newOutline[index].title = e.target.value
                          store.setOutline(newOutline)
                        }}
                        disabled={store.isGenerating}
                      />
                      {chapter.action === 'new' && (
                        <span className="rounded border border-[color-mix(in_srgb,var(--success)_22%,var(--border))] bg-[var(--success-soft)] px-2 py-0.5 text-xs font-medium text-[var(--success)]">新增</span>
                      )}
                      {chapter.action === 'regenerate' && (
                        <span className="rounded border border-[color-mix(in_srgb,var(--warning)_22%,var(--border))] bg-[var(--warning-soft)] px-2 py-0.5 text-xs font-medium text-[var(--warning)]">更新</span>
                      )}
                      {chapter.action === 'skip' && (
                        <span className="rounded border border-border bg-secondary px-2 py-0.5 text-xs font-medium text-muted-foreground">保留</span>
                      )}
                    </div>
                    <textarea
                        className="-ml-2 h-20 w-full resize-none rounded bg-transparent px-2 py-1 text-sm text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                        value={chapter.summary}
                        onChange={(e) => {
                          if (!store.outline) return;
                          const newOutline = [...store.outline]
                          newOutline[index].summary = e.target.value
                          if (newOutline[index].action === 'skip') {
                            newOutline[index].action = 'regenerate'
                          }
                          store.setOutline(newOutline)
                        }}
                        disabled={store.isGenerating}
                      />
                  </div>
                  <div className="flex-shrink-0 flex flex-col gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-8 w-8 text-muted-foreground hover:text-foreground"
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
                      className="h-8 w-8 text-muted-foreground hover:text-foreground"
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
              className="w-full max-w-md border-dashed bg-card text-muted-foreground hover:border-solid hover:bg-secondary/50"
              disabled={store.isGenerating}
              onClick={() => {
                if (!store.outline) return;
                store.setOutline([
                  ...store.outline,
                  {
                    title: '新章节标题',
                    summary: '请在此输入章节概要，指导 AI 生成内容...',
                    sort: store.outline.length + 1,
                    files: [],
                    action: 'new'
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
