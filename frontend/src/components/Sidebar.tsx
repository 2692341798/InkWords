import { useEffect, useState } from 'react'
import JSZip from 'jszip'
import { useStreamStore } from '@/store/streamStore'
import { useBlogStore } from '@/store/blogStore'
import type { BlogNode } from '@/store/blogStore'
import { GitBranch, CheckCircle2, CircleDashed, Loader2, BookOpen, ChevronRight, ChevronDown, Plus, LogOut, FolderArchive, Square, CheckSquare, RefreshCw } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from './ui/button'

export function Sidebar() {
  const streamStore = useStreamStore()
  const { blogs, fetchBlogs, selectedBlog, selectBlog } = useBlogStore()
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set())
  const [isBatchMode, setIsBatchMode] = useState(false)
  const [selectedForExport, setSelectedForExport] = useState<Set<string>>(new Set())
  const [isExporting, setIsExporting] = useState(false)

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

  const toggleNodeSelection = (node: BlogNode) => {
    const newSelected = new Set(selectedForExport)
    const isCurrentlySelected = newSelected.has(node.id)
    
    const setSelection = (n: BlogNode, select: boolean) => {
      if (select) newSelected.add(n.id)
      else newSelected.delete(n.id)
      if (n.children) {
        n.children.forEach(child => setSelection(child, select))
      }
    }
    
    setSelection(node, !isCurrentlySelected)
    setSelectedForExport(newSelected)
  }

  const handleBatchExport = async () => {
    if (selectedForExport.size === 0) return
    setIsExporting(true)
    try {
      const zip = new JSZip()
      
      const addNodeToZip = (node: BlogNode, folder: JSZip | null, index: number) => {
        if (selectedForExport.has(node.id)) {
          const title = node.title || `未命名_${index}`
          const filename = `${title}.md`
          
          if (node.children && node.children.length > 0) {
            const subFolder = folder ? folder.folder(title) : zip.folder(title)
            node.children.forEach((child, idx) => addNodeToZip(child, subFolder, idx))
          } else {
            const targetFolder = folder || zip
            const prefix = node.chapter_sort > 0 ? `${String(node.chapter_sort).padStart(2, '0')}-` : ''
            targetFolder.file(`${prefix}${filename}`, `# ${title}\n\n${node.content || ''}`)
          }
        } else {
          if (node.children && node.children.length > 0) {
            node.children.forEach((child, idx) => addNodeToZip(child, folder, idx))
          }
        }
      }

      blogs.forEach((blog, idx) => addNodeToZip(blog, null, idx))
      
      const blob = await zip.generateAsync({ type: 'blob' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'blogs_export.zip'
      document.body.appendChild(a)
      a.click()
      URL.revokeObjectURL(url)
      document.body.removeChild(a)
      
      setIsBatchMode(false)
      setSelectedForExport(new Set())
    } catch (err) {
      console.error('Failed to export batch zip:', err)
    } finally {
      setIsExporting(false)
    }
  }

  const renderBlogNode = (node: BlogNode, level = 0) => {
    const isExpanded = expandedNodes.has(node.id)
    const hasChildren = node.children && node.children.length > 0
    const isSelected = selectedBlog?.id === node.id
    const isExportSelected = selectedForExport.has(node.id)

    return (
      <div key={node.id} className="flex flex-col">
        <div
          className={cn(
            "flex items-center py-2 px-3 hover:bg-zinc-100 rounded-md cursor-pointer text-sm gap-2 transition-colors",
            isSelected && !isBatchMode ? "bg-indigo-50 text-indigo-700 font-medium" : "text-zinc-700"
          )}
          style={{ paddingLeft: `${level * 16 + 12}px` }}
          onClick={() => {
            if (isBatchMode) {
              toggleNodeSelection(node)
            } else {
              if (hasChildren) {
                toggleNode(node.id)
              } else {
                selectBlog(node)
              }
            }
          }}
        >
          {isBatchMode && (
            <div className="shrink-0 text-indigo-500" onClick={(e) => {
               e.stopPropagation()
               toggleNodeSelection(node)
            }}>
              {isExportSelected ? <CheckSquare className="w-4 h-4" /> : <Square className="w-4 h-4 text-zinc-300" />}
            </div>
          )}
          {hasChildren ? (
            <div 
              className="w-4 h-4 flex items-center justify-center text-zinc-400"
              onClick={(e) => {
                if (isBatchMode) {
                  e.stopPropagation()
                  toggleNode(node.id)
                }
              }}
            >
              {isExpanded ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
            </div>
          ) : (
            <BookOpen className={cn("w-4 h-4", isSelected && !isBatchMode ? "text-indigo-500" : "text-zinc-400")} />
          )}
          <span className="truncate flex-1 min-w-0">{node.title || '无标题'}</span>
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
      
      <div className="flex-1 flex flex-col min-h-0 overflow-hidden">
        {/* Stream / Generator Outline Section */}
        {streamStore.outline && streamStore.outline.length > 0 && (
          <div className="p-4 border-b border-zinc-100">
            <div className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-4 flex items-center justify-between">
              <span>当前生成任务</span>
              <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => selectBlog(null)}>
                <Plus className="h-4 w-4" />
              </Button>
            </div>
            <div className="space-y-3 max-h-[30vh] overflow-y-auto custom-scrollbar">
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
                        let found: BlogNode | null = null
                        let parentIdToExpand: string | null = null

                        const searchNested = (nodes: BlogNode[], parentId: string | null = null) => {
                          for (const node of nodes) {
                            if (node.title === ch.title && node.chapter_sort === ch.sort) {
                              found = node
                              parentIdToExpand = parentId
                              return
                            }
                            if (node.children && node.children.length > 0) {
                              searchNested(node.children, node.id)
                              if (found) return
                            }
                          }
                        }

                        searchNested(blogs)

                        if (found) {
                          if (parentIdToExpand) {
                            const newExpanded = new Set(expandedNodes)
                            newExpanded.add(parentIdToExpand)
                            setExpandedNodes(newExpanded)
                          }
                          selectBlog(found)
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
        <div className="flex-1 flex flex-col min-h-0 overflow-hidden relative">
          <div className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-4 flex items-center justify-between shrink-0 px-4 pt-4">
            <span>历史博客</span>
            <div className="flex items-center gap-1">
              <Button 
                variant={isBatchMode ? "secondary" : "ghost"} 
                size="icon" 
                className={cn("h-6 w-6", isBatchMode && "bg-indigo-100 text-indigo-700 hover:bg-indigo-200")} 
                onClick={() => {
                  setIsBatchMode(!isBatchMode)
                  setSelectedForExport(new Set())
                }}
                title="批量导出"
              >
                <FolderArchive className="w-3.5 h-3.5" />
              </Button>
              <Button variant="ghost" size="icon" className="h-6 w-6" onClick={fetchBlogs} title="刷新列表">
                <RefreshCw className="w-3.5 h-3.5" />
              </Button>
            </div>
          </div>
          
          <div className={cn("flex-1 overflow-y-auto custom-scrollbar flex flex-col space-y-1 px-4", isBatchMode ? "pb-16" : "pb-4")}>
            {blogs.length > 0 ? (
              blogs.map(blog => renderBlogNode(blog))
            ) : (
              <div className="text-sm text-zinc-400 text-center py-6">
                暂无历史记录
              </div>
            )}
          </div>

          {/* Batch Export Action Bar */}
          {isBatchMode && (
            <div className="absolute bottom-0 left-0 right-0 bg-white border-t border-zinc-200 p-3 shadow-[0_-4px_6px_-1px_rgba(0,0,0,0.05)] flex items-center justify-between">
              <span className="text-xs text-zinc-500">已选 {selectedForExport.size} 项</span>
              <div className="flex items-center gap-2">
                <Button variant="ghost" size="sm" className="h-7 text-xs" onClick={() => setIsBatchMode(false)}>取消</Button>
                <Button 
                  variant="default" 
                  size="sm" 
                  className="h-7 text-xs bg-indigo-600 hover:bg-indigo-700"
                  onClick={handleBatchExport}
                  disabled={selectedForExport.size === 0 || isExporting}
                >
                  {isExporting ? <Loader2 className="w-3 h-3 animate-spin mr-1" /> : null}
                  导出 ZIP
                </Button>
              </div>
            </div>
          )}
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
