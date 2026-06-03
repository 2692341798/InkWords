import { useMemo } from 'react'
import { CheckCircle2, CircleDashed, Loader2, Plus } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '../ui/button'
import { useBlogStore } from '@/store/blogStore'
import type { BlogNode } from '@/store/blogStore'
import type { Chapter } from '@/store/streamStore'

type BlogMapValue = { node: BlogNode; parentId: string | null }

type StreamOutlineSectionProps = {
  outline: Chapter[]
  chapterStatus: Record<number, 'pending' | 'generating' | 'completed' | 'error'>
  chapterErrors: Record<number, string>
  parentBlogId: string | null
  isGenerating: boolean
  isAnalyzing: boolean
  resetStream: () => void
  setCurrentView: (view: 'generator' | 'dashboard') => void
  blogs: BlogNode[]
  fetchBlogs: () => Promise<void>
  selectBlog: (blog: BlogNode) => void
  blogMap: Map<string, BlogMapValue>
  expandedNodes: Set<string>
  setExpandedNodes: (next: Set<string>) => void
}

export function StreamOutlineSection(props: StreamOutlineSectionProps) {
  const {
    outline,
    chapterStatus,
    chapterErrors,
    parentBlogId,
    isGenerating,
    isAnalyzing,
    resetStream,
    setCurrentView,
    blogs,
    fetchBlogs,
    selectBlog,
    blogMap,
    expandedNodes,
    setExpandedNodes,
  } = props

  const hasOutline = outline && outline.length > 0

  const canReset = useMemo(() => isGenerating || isAnalyzing, [isAnalyzing, isGenerating])

  if (!hasOutline) return null

  return (
    <div className="p-4 border-b border-zinc-100">
      <div className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-4 flex items-center justify-between">
        <span>当前生成任务</span>
        <Button
          variant="ghost"
          size="icon"
          className="h-6 w-6"
          onClick={() => {
            if (canReset) {
              if (window.confirm('当前有任务正在执行，确定要终止并开启新工作区吗？')) {
                resetStream()
                setCurrentView('generator')
              }
            } else {
              resetStream()
              setCurrentView('generator')
            }
          }}
          title="新建工作区"
        >
          <Plus className="h-4 w-4" />
        </Button>
      </div>

      <div className="space-y-3 max-h-[30vh] overflow-y-auto custom-scrollbar">
        {chapterStatus[0] && (
          <div
            key={0}
            className={cn(
              "p-3 rounded-lg border flex items-start gap-3 transition-colors",
              chapterStatus[0] === 'completed'
                ? "bg-green-50/50 border-green-100 hover:bg-green-50 cursor-pointer"
                : chapterStatus[0] === 'generating'
                  ? "bg-indigo-50 border-indigo-200"
                  : "bg-zinc-50 border-zinc-100",
            )}
            onClick={() => {
              if (chapterStatus[0] === 'completed') {
                const id = parentBlogId
                if (id) {
                  const found = blogs.find(b => b.id === id)
                  if (found) {
                    selectBlog(found)
                  } else {
                    fetchBlogs().then(() => {
                      const updatedBlogs = useBlogStore.getState().blogs
                      const newFound = updatedBlogs.find(b => b.id === id)
                      if (newFound) selectBlog(newFound)
                    })
                  }
                }
              }
            }}
          >
            <div className="mt-0.5">
              {chapterStatus[0] === 'completed' ? (
                <CheckCircle2 className="w-4 h-4 text-green-500" />
              ) : chapterStatus[0] === 'generating' ? (
                <Loader2 className="w-4 h-4 text-indigo-500 animate-spin" />
              ) : (
                <CircleDashed className="w-4 h-4 text-zinc-400" />
              )}
            </div>
            <div className="flex-1">
              <div className="text-sm font-medium text-zinc-800">系列导读</div>
                <div className="text-xs text-zinc-500 mt-1 line-clamp-2">
                  {chapterStatus[0] === 'error' && chapterErrors[0] ? `失败原因：${chapterErrors[0]}` : '全系列的引言与总结概览'}
                </div>
            </div>
          </div>
        )}

        {outline.map((ch) => {
          const status = chapterStatus[ch.sort]
          return (
            <div
              key={ch.sort}
              className={cn(
                "p-3 rounded-lg border flex items-start gap-3 transition-colors",
                status === 'completed'
                  ? "bg-green-50/50 border-green-100 hover:bg-green-50 cursor-pointer"
                  : status === 'generating'
                    ? "bg-indigo-50 border-indigo-200"
                    : "bg-zinc-50 border-zinc-100",
              )}
              onClick={() => {
                if (status === 'completed') {
                  const key = `${ch.title}_${ch.sort}`
                  const found = blogMap.get(key)
                  if (found) {
                    if (found.parentId) {
                      const newExpanded = new Set(expandedNodes)
                      newExpanded.add(found.parentId)
                      setExpandedNodes(newExpanded)
                    }
                    selectBlog(found.node)
                  }
                }
              }}
            >
              <div className="mt-0.5">
                {status === 'completed' ? (
                  <CheckCircle2 className="w-4 h-4 text-green-500" />
                ) : status === 'generating' ? (
                  <Loader2 className="w-4 h-4 text-indigo-500 animate-spin" />
                ) : (
                  <CircleDashed className="w-4 h-4 text-zinc-400" />
                )}
              </div>
              <div className="flex-1">
                <div className="text-sm font-medium text-zinc-800">{ch.title}</div>
                <div className="text-xs text-zinc-500 mt-1 line-clamp-2">
                  {status === 'error' && chapterErrors[ch.sort] ? `失败原因：${chapterErrors[ch.sort]}` : ch.summary}
                </div>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
