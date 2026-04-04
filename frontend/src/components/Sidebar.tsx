import { useEffect, useState } from 'react'
import { useStreamStore } from '@/store/streamStore'
import { useBlogStore } from '@/store/blogStore'
import type { BlogNode } from '@/store/blogStore'
import { GitBranch, CheckCircle2, CircleDashed, Loader2, BookOpen, ChevronRight, ChevronDown, Plus, LogOut } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from './ui/button'

export function Sidebar() {
  const streamStore = useStreamStore()
  const { blogs, fetchBlogs, selectedBlog, selectBlog } = useBlogStore()
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set())

  useEffect(() => {
    fetchBlogs()
  }, [fetchBlogs])

  const toggleNode = (id: string) => {
    const newExpanded = new Set(expandedNodes)
    if (newExpanded.has(id)) {
      newExpanded.delete(id)
    } else {
      newExpanded.add(id)
    }
    setExpandedNodes(newExpanded)
  }

  const renderBlogNode = (node: BlogNode, level = 0) => {
    const isExpanded = expandedNodes.has(node.id)
    const hasChildren = node.children && node.children.length > 0
    const isSelected = selectedBlog?.id === node.id

    return (
      <div key={node.id} className="flex flex-col">
        <div
          className={cn(
            "flex items-center py-2 px-3 hover:bg-zinc-100 rounded-md cursor-pointer text-sm gap-2 transition-colors",
            isSelected ? "bg-indigo-50 text-indigo-700 font-medium" : "text-zinc-700"
          )}
          style={{ paddingLeft: `${level * 16 + 12}px` }}
          onClick={() => {
            if (hasChildren) {
              toggleNode(node.id)
            } else {
              selectBlog(node)
            }
          }}
        >
          {hasChildren ? (
            <div className="w-4 h-4 flex items-center justify-center text-zinc-400">
              {isExpanded ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
            </div>
          ) : (
            <BookOpen className={cn("w-4 h-4", isSelected ? "text-indigo-500" : "text-zinc-400")} />
          )}
          <span className="truncate flex-1">{node.title || '无标题'}</span>
        </div>
        
        {hasChildren && isExpanded && (
          <div className="flex flex-col">
            {node.children.map(child => renderBlogNode(child, level + 1))}
          </div>
        )}
      </div>
    )
  }

  return (
    <div className="w-80 bg-white border-r border-zinc-200 flex flex-col print:hidden">
      <div className="p-4 border-b border-zinc-200 flex flex-col gap-4 shrink-0">
        <div className="flex items-center gap-2 font-semibold text-lg text-zinc-800">
          <GitBranch className="w-5 h-5 text-indigo-600" />
          墨言博客助手
        </div>
        <Button 
          className="w-full gap-2 bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm"
          onClick={() => selectBlog(null)}
        >
          <Plus className="w-4 h-4" />
          返回
        </Button>
      </div>
      
      <div className="flex-1 overflow-y-auto flex flex-col">
        {/* Stream / Generator Outline Section */}
        {streamStore.outline && streamStore.outline.length > 0 && (
          <div className="p-4 border-b border-zinc-100">
            <div className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-4 flex items-center justify-between">
              <span>当前生成任务</span>
              <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => selectBlog(null)}>
                <Plus className="h-4 w-4" />
              </Button>
            </div>
            <div className="space-y-3">
              {streamStore.outline.map((ch) => {
                const status = streamStore.chapterStatus[ch.sort]
                return (
                  <div 
                    key={ch.sort} 
                    className={cn(
                      "p-3 rounded-lg border flex items-start gap-3 transition-colors",
                      status === 'completed'
                        ? "bg-green-50/50 border-green-100 hover:bg-green-50 cursor-pointer" 
                        : status === 'generating'
                          ? "bg-indigo-50 border-indigo-200"
                          : "bg-zinc-50 border-zinc-100"
                    )}
                    onClick={() => {
                      if (status === 'completed') {
                        // Find the matching real blog from history
                        // We assume the generated ones are at the top, we match by title
                        const matchingBlog = blogs.find(b => b.title === ch.title && b.chapter_sort === ch.sort)
                        if (matchingBlog) {
                          selectBlog(matchingBlog)
                        } else {
                          // fallback if parent-child structure makes it nested
                          let found: BlogNode | null = null
                          const searchNested = (nodes: BlogNode[]) => {
                            for (const node of nodes) {
                              if (node.title === ch.title && node.chapter_sort === ch.sort) {
                                found = node
                                return
                              }
                              if (node.children) searchNested(node.children)
                            }
                          }
                          searchNested(blogs)
                          if (found) {
                            selectBlog(found)
                          }
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
                      <div className="text-xs text-zinc-500 mt-1 line-clamp-2">{ch.summary}</div>
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        )}

        {/* History Blogs Section */}
        <div className="p-4 flex-1">
          <div className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-4 flex items-center justify-between">
            <span>历史博客</span>
            <Button variant="ghost" size="icon" className="h-6 w-6" onClick={fetchBlogs}>
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="text-zinc-500"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/></svg>
            </Button>
          </div>
          
          <div className="flex flex-col space-y-1">
            {blogs.length > 0 ? (
              blogs.map(blog => renderBlogNode(blog))
            ) : (
              <div className="text-sm text-zinc-400 text-center py-6">
                暂无历史记录
              </div>
            )}
          </div>
        </div>
      </div>

      <div className="p-4 border-t border-zinc-200 mt-auto shrink-0">
        <Button
          variant="ghost"
          className="w-full flex items-center justify-start gap-2 text-zinc-600 hover:text-red-600 hover:bg-red-50"
          onClick={() => {
            localStorage.removeItem('token')
            window.location.href = '/'
          }}
        >
          <LogOut className="w-4 h-4" />
          退出登录
        </Button>
      </div>
    </div>
  )
}
