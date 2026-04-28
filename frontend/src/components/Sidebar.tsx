import { useEffect, useMemo, useState } from 'react'
import JSZip from 'jszip'
import { useStreamStore } from '@/store/streamStore'
import { useBlogStore } from '@/store/blogStore'
import type { BlogNode } from '@/store/blogStore'
import { BookOpen, CheckCircle2, CheckSquare, ChevronDown, ChevronRight, CircleDashed, Download, FileArchive, FolderArchive, GitBranch, Loader2, LogOut, Plus, RefreshCw, Sparkles, Square, Trash2, User } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from './ui/button'
import { ConfirmDialog } from './ui/confirm-dialog'
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from './ui/dropdown-menu'
import { toast } from 'sonner'

export function Sidebar() {
  const streamStore = useStreamStore()
  const { blogs, fetchBlogs, selectedBlog, selectBlog, currentView, setCurrentView } = useBlogStore()
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set())
  const [isBatchMode, setIsBatchMode] = useState(false)
  const [selectedForExport, setSelectedForExport] = useState<Set<string>>(new Set())
  const [isExporting, setIsExporting] = useState(false)
  const [isExportingPDF, setIsExportingPDF] = useState(false)
  const [isSyncingSeriesToObsidian, setIsSyncingSeriesToObsidian] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const [showBatchDeleteConfirm, setShowBatchDeleteConfirm] = useState(false)

  const blogMap = useMemo(() => {
    const map = new Map<string, { node: BlogNode; parentId: string | null }>()
    const traverse = (nodes: BlogNode[], parentId: string | null = null) => {
      nodes.forEach(node => {
        const key = `${node.title}_${node.chapter_sort}`
        map.set(key, { node, parentId })
        if (node.children) {
          traverse(node.children, node.id)
        }
      })
    }
    traverse(blogs)
    return map
  }, [blogs])

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

  const selectedSeriesRoots = useMemo(() => {
    return blogs.filter(b => Boolean(b.children?.length) && selectedForExport.has(b.id))
  }, [blogs, selectedForExport])

  const handleBatchExportSeriesToObsidian = async () => {
    if (selectedSeriesRoots.length === 0) {
      toast.error('请先选择一个系列父节点')
      return
    }

    setIsSyncingSeriesToObsidian(true)
    const token = localStorage.getItem('token')

    try {
      toast.loading('正在同步系列到 Obsidian...', { id: 'sync-series-obsidian' })

      for (const series of selectedSeriesRoots) {
        const res = await fetch(`/api/v1/blogs/${series.id}/export/obsidian/series`, {
          method: 'POST',
          headers: {
            ...(token ? { 'Authorization': `Bearer ${token}` } : {})
          }
        })

        const data = await res.json().catch(() => null)
        if (!res.ok || data?.code !== 200) {
          throw new Error(data?.message || '同步系列失败')
        }
      }

      toast.success(`成功同步 ${selectedSeriesRoots.length} 个系列到 Obsidian`, { id: 'sync-series-obsidian' })
      setIsBatchMode(false)
      setSelectedForExport(new Set())
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : '同步系列失败'
      toast.error(message, { id: 'sync-series-obsidian' })
    } finally {
      setIsSyncingSeriesToObsidian(false)
    }
  }

  const sanitizeDownloadFilename = (name: string) => {
    return (name || 'series')
      .replaceAll('/', '-')
      .replaceAll('\\', '-')
      .replaceAll(':', '：')
      .trim()
  }

  const handleBatchExportSeriesPDF = async () => {
    if (selectedSeriesRoots.length === 0) {
      toast.error('请先选择一个系列父节点')
      return
    }

    const token = localStorage.getItem('token')
    setIsExportingPDF(true)

    try {
      toast.loading(`正在导出 PDF：0/${selectedSeriesRoots.length}`, { id: 'export-series-pdf' })

      let done = 0
      for (const series of selectedSeriesRoots) {
        try {
          const res = await fetch(`/api/v1/blogs/${series.id}/export/pdf`, {
            headers: {
              ...(token ? { 'Authorization': `Bearer ${token}` } : {})
            }
          })

          if (!res.ok) {
            const data = await res.json().catch(() => null)
            throw new Error(data?.message || '导出失败')
          }

          const blob = await res.blob()
          const url = URL.createObjectURL(blob)
          const a = document.createElement('a')
          a.href = url
          a.download = `${sanitizeDownloadFilename(series.title)}.pdf`
          document.body.appendChild(a)
          a.click()
          URL.revokeObjectURL(url)
          document.body.removeChild(a)
        } catch (err: unknown) {
          const message = err instanceof Error ? err.message : '导出失败'
          toast.error(`《${series.title || '未命名系列'}》导出失败：${message}`)
        } finally {
          done += 1
          toast.loading(`正在导出 PDF：${done}/${selectedSeriesRoots.length}`, { id: 'export-series-pdf' })
        }
      }

      toast.success(`已开始下载 ${selectedSeriesRoots.length} 份 PDF`, { id: 'export-series-pdf' })
    } finally {
      setIsExportingPDF(false)
    }
  }

  const executeBatchDelete = async () => {
    setIsDeleting(true)
    setShowBatchDeleteConfirm(false)
    try {
      const { batchDeleteBlogs } = useBlogStore.getState()
      await batchDeleteBlogs(Array.from(selectedForExport))
      
      setIsBatchMode(false)
      setSelectedForExport(new Set())
    } catch (err) {
      console.error('Failed to batch delete blogs:', err)
      alert('批量删除失败，请稍后重试')
    } finally {
      setIsDeleting(false)
    }
  }

  const handleBatchDeleteClick = () => {
    if (selectedForExport.size === 0 || isDeleting) return
    setShowBatchDeleteConfirm(true)
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
            isSelected && !hasChildren && !isBatchMode ? "bg-indigo-50 text-indigo-700 font-medium" : "text-zinc-700"
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
              className="w-4 h-4 flex items-center justify-center text-zinc-400 hover:text-zinc-600 rounded-sm hover:bg-zinc-200"
              onClick={(e) => {
                e.stopPropagation()
                toggleNode(node.id)
              }}
            >
              {isExpanded ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
            </div>
          ) : (
            <BookOpen className={cn("w-4 h-4", isSelected && !isBatchMode ? "text-indigo-500" : "text-zinc-400")} />
          )}
          <span className="truncate flex-1 min-w-0" title={node.title || '无标题'}>{node.title || '无标题'}</span>
        </div>
        
        {hasChildren && isExpanded && (
          <div className="flex flex-col">
            {/* If node is a parent, manually inject the parent node itself as the first "Series Guide" child in the UI */}
            {node.parent_id === null && (
              <div
                className={cn(
                  "flex items-center py-2 px-3 hover:bg-zinc-100 rounded-md cursor-pointer text-sm gap-2 transition-colors",
                  selectedBlog?.id === node.id && !isBatchMode ? "bg-indigo-50 text-indigo-700 font-medium" : "text-zinc-700"
                )}
                style={{ paddingLeft: `${(level + 1) * 16 + 12}px` }}
                onClick={() => {
                  if (isBatchMode) {
                    toggleNodeSelection(node)
                  } else {
                    selectBlog(node)
                  }
                }}
              >
                <BookOpen className={cn("w-4 h-4", selectedBlog?.id === node.id && !isBatchMode ? "text-indigo-500" : "text-zinc-400")} />
                <span className="truncate flex-1 min-w-0" title="系列导读 (概览)">系列导读 (概览)</span>
              </div>
            )}
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
          onClick={() => {
            if (streamStore.isGenerating || streamStore.isAnalyzing) {
              if (window.confirm('当前有任务正在执行，确定要终止并开启新工作区吗？')) {
                streamStore.reset()
                setCurrentView('generator')
              }
            } else {
              streamStore.reset()
              setCurrentView('generator')
            }
          }}
        >
          <Plus className="w-4 h-4" />
          新工作区
        </Button>
      </div>
      
      <div className="flex-1 flex flex-col min-h-0 overflow-hidden">
        {/* Stream / Generator Outline Section */}
        {streamStore.outline && streamStore.outline.length > 0 && (
          <div className="p-4 border-b border-zinc-100">
            <div className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-4 flex items-center justify-between">
              <span>当前生成任务</span>
              <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => {
                if (streamStore.isGenerating || streamStore.isAnalyzing) {
                  if (window.confirm('当前有任务正在执行，确定要终止并开启新工作区吗？')) {
                    streamStore.reset()
                    setCurrentView('generator')
                  }
                } else {
                  streamStore.reset()
                  setCurrentView('generator')
                }
              }} title="新建工作区">
                <Plus className="h-4 w-4" />
              </Button>
            </div>
            <div className="space-y-3 max-h-[30vh] overflow-y-auto custom-scrollbar">
              {/* 系列导读 */}
              {streamStore.chapterStatus[0] && (
                  <div 
                    key={0} 
                    className={cn(
                      "p-3 rounded-lg border flex items-start gap-3 transition-colors",
                      streamStore.chapterStatus[0] === 'completed'
                        ? "bg-green-50/50 border-green-100 hover:bg-green-50 cursor-pointer" 
                        : streamStore.chapterStatus[0] === 'generating'
                          ? "bg-indigo-50 border-indigo-200"
                          : "bg-zinc-50 border-zinc-100"
                    )}
                    onClick={() => {
                      if (streamStore.chapterStatus[0] === 'completed') {
                        const parentBlogId = streamStore.parentBlogId
                        if (parentBlogId) {
                          const found = blogs.find(b => b.id === parentBlogId)
                          if (found) {
                            selectBlog(found)
                          } else {
                            fetchBlogs().then(() => {
                               const updatedBlogs = useBlogStore.getState().blogs
                               const newFound = updatedBlogs.find(b => b.id === parentBlogId)
                               if (newFound) selectBlog(newFound)
                            })
                          }
                        }
                      }
                    }}
                  >
                    <div className="mt-0.5">
                      {streamStore.chapterStatus[0] === 'completed' ? (
                        <CheckCircle2 className="w-4 h-4 text-green-500" />
                      ) : streamStore.chapterStatus[0] === 'generating' ? (
                        <Loader2 className="w-4 h-4 text-indigo-500 animate-spin" />
                      ) : (
                        <CircleDashed className="w-4 h-4 text-zinc-400" />
                      )}
                    </div>
                    <div className="flex-1">
                      <div className="text-sm font-medium text-zinc-800">系列导读</div>
                      <div className="text-xs text-zinc-500 mt-1 line-clamp-2">全系列的引言与总结概览</div>
                    </div>
                  </div>
              )}
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
          
          <div className={cn("flex-1 overflow-y-auto custom-scrollbar flex flex-col space-y-1 px-4", isBatchMode ? "pb-24" : "pb-4")}>
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
            <div className="absolute bottom-0 left-0 right-0 bg-white border-t border-zinc-200 p-3 shadow-[0_-4px_6px_-1px_rgba(0,0,0,0.05)] flex flex-col gap-2">
              <div className="flex items-center justify-between gap-2">
                <span className="text-xs text-zinc-500">已选 {selectedForExport.size} 项</span>
                <Button variant="ghost" size="sm" onClick={() => setIsBatchMode(false)}>取消</Button>
              </div>
              <div className="flex items-center justify-end gap-2">
                <DropdownMenu>
                  <DropdownMenuTrigger
                    render={
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={isExporting || isDeleting || isSyncingSeriesToObsidian || isExportingPDF}
                      >
                        <Download data-icon="inline-start" />
                        导出 / 同步
                        <ChevronDown data-icon="inline-end" className="opacity-60" />
                      </Button>
                    }
                  />
                  <DropdownMenuContent align="end" className="w-56">
                    <DropdownMenuItem
                      onClick={handleBatchExportSeriesPDF}
                      disabled={selectedSeriesRoots.length === 0 || isExporting || isDeleting || isSyncingSeriesToObsidian || isExportingPDF}
                      className="cursor-pointer"
                    >
                      <Download data-icon="inline-start" className="text-muted-foreground" />
                      <span>导出系列 PDF</span>
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      onClick={handleBatchExport}
                      disabled={selectedForExport.size === 0 || isExporting || isDeleting}
                      className="cursor-pointer"
                    >
                      <FileArchive data-icon="inline-start" className="text-muted-foreground" />
                      <span>导出 ZIP</span>
                    </DropdownMenuItem>
                    <DropdownMenuSeparator className="my-1" />
                    <DropdownMenuItem
                      onClick={handleBatchExportSeriesToObsidian}
                      disabled={selectedSeriesRoots.length === 0 || isExporting || isDeleting || isSyncingSeriesToObsidian || isExportingPDF}
                      className="cursor-pointer"
                    >
                      {isSyncingSeriesToObsidian ? (
                        <Loader2 data-icon="inline-start" className="animate-spin" />
                      ) : (
                        <Sparkles data-icon="inline-start" className="text-muted-foreground" />
                      )}
                      <span>同步系列到 Obsidian</span>
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>

                <Button 
                  type="button"
                  variant="destructive" 
                  size="icon-sm"
                  onClick={(e) => {
                    e.preventDefault()
                    e.stopPropagation()
                    handleBatchDeleteClick()
                  }}
                  disabled={selectedForExport.size === 0 || isDeleting || isExporting || isExportingPDF || isSyncingSeriesToObsidian}
                  title="批量删除"
                >
                  {isDeleting ? <Loader2 className="animate-spin" /> : <Trash2 />}
                </Button>
              </div>
            </div>
          )}
        </div>
      </div>

      <div className="p-4 border-t border-zinc-200 mt-auto shrink-0 flex flex-col gap-2">
        <Button
          variant="ghost"
          className={cn(
            "w-full flex items-center justify-start gap-2",
            currentView === 'dashboard' && !selectedBlog ? "bg-indigo-50 text-indigo-700 font-medium" : "text-zinc-600 hover:bg-zinc-100 hover:text-zinc-900"
          )}
          onClick={() => setCurrentView('dashboard')}
        >
          <User className="w-4 h-4" />
          个人中心
        </Button>
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

      <ConfirmDialog
        isOpen={showBatchDeleteConfirm}
        title="确认批量删除"
        message={`确定要删除选中的 ${selectedForExport.size} 篇博客吗？此操作不可恢复。`}
        confirmText="确认删除"
        onConfirm={executeBatchDelete}
        onCancel={() => setShowBatchDeleteConfirm(false)}
        isDestructive={true}
      />
    </div>
  )
}
