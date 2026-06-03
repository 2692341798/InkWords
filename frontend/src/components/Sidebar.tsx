import { useEffect, useMemo, useState } from 'react'
import { useStreamStore } from '@/store/streamStore'
import { useBlogStore } from '@/store/blogStore'
import { useReviewStore } from '@/store/reviewStore'
import type { BlogNode } from '@/store/blogStore'
import { BookOpen, CheckSquare, ChevronDown, ChevronRight, FilePenLine, FolderArchive, GitBranch, Loader2, LogOut, Plus, RefreshCw, Sparkles, Square, User } from 'lucide-react'
import { syncExpandedNodesWithSelection } from '@/lib/blogTreeSelection'
import { cn } from '@/lib/utils'
import { Button } from './ui/button'
import { ConfirmDialog } from './ui/confirm-dialog'
import { toast } from 'sonner'
import { authTokenStore } from '@/lib/authTokenStore'
import { useBatchExportZip } from '@/hooks/useBatchExportZip'
import { exportSeriesPdfs, syncSeriesToObsidian } from '@/services/sidebarExport'
import { useSidebarBatchSelection } from '@/hooks/useSidebarBatchSelection'
import { useShallow } from 'zustand/react/shallow'
import { SidebarBatchActionBar } from './sidebar/SidebarBatchActionBar'
import { SidebarShell } from './sidebar/SidebarShell'
import { StreamOutlineSection } from './sidebar/StreamOutlineSection'

export function Sidebar() {
  const streamStore = useStreamStore(
    useShallow((state) => ({
      outline: state.outline,
      chapterStatus: state.chapterStatus,
      chapterErrors: state.chapterErrors,
      parentBlogId: state.parentBlogId,
      isGenerating: state.isGenerating,
      isAnalyzing: state.isAnalyzing,
      reset: state.reset,
    })),
  )
  const { blogs, fetchBlogs, createDraftBlog, selectedBlog, selectBlog, currentView, setCurrentView } = useBlogStore()
  const setShouldResumeSessionOnOpen = useReviewStore((state) => state.setShouldResumeSessionOnOpen)
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set())
  const [isSyncingSeriesToObsidian, setIsSyncingSeriesToObsidian] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const [isExportingPDF, setIsExportingPDF] = useState(false)
  const [showBatchDeleteConfirm, setShowBatchDeleteConfirm] = useState(false)
  const [isCreatingDraft, setIsCreatingDraft] = useState(false)
  const [showWorkspaceResetConfirm, setShowWorkspaceResetConfirm] = useState(false)
  const [showLogoutConfirm, setShowLogoutConfirm] = useState(false)
  const {
    isBatchMode,
    selectedForExport,
    selectedSeriesRoots,
    toggleNodeSelection,
    toggleBatchMode,
    closeBatchMode,
    resetBatchSelection,
  } = useSidebarBatchSelection(blogs)

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

  useEffect(() => {
    // Why: 系列父博客在生成完成后会被自动选中，如果不同步展开状态，
    // 左侧历史树看起来就像只有一篇导读，用户看不到刚生成的子文章。
    setExpandedNodes((previous) => syncExpandedNodesWithSelection(previous, selectedBlog))
  }, [selectedBlog])

  const toggleNode = (id: string) => {
    const newExpanded = new Set(expandedNodes)
    if (newExpanded.has(id)) {
      newExpanded.delete(id)
    } else {
      newExpanded.add(id)
    }
    setExpandedNodes(newExpanded)
  }

  const { isExporting, handleBatchExport } = useBatchExportZip({
    blogs,
    selectedForExport,
    onDone: () => {
      resetBatchSelection()
    },
  })

  const handleBatchExportSeriesToObsidian = async () => {
    if (selectedSeriesRoots.length === 0) {
      toast.error('请先选择一个系列父节点')
      return
    }

    setIsSyncingSeriesToObsidian(true)

    try {
      toast.loading('正在同步系列到 Obsidian...', { id: 'sync-series-obsidian' })
      const syncedCount = await syncSeriesToObsidian(selectedSeriesRoots)

      toast.success(`成功同步 ${syncedCount} 个系列到 Obsidian`, { id: 'sync-series-obsidian' })
      resetBatchSelection()
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : '同步系列失败'
      toast.error(message, { id: 'sync-series-obsidian' })
    } finally {
      setIsSyncingSeriesToObsidian(false)
    }
  }

  const handleBatchExportSeriesPDF = async () => {
    if (selectedSeriesRoots.length === 0) {
      toast.error('请先选择一个系列父节点')
      return
    }

    setIsExportingPDF(true)

    try {
      toast.loading(`正在导出 PDF：0/${selectedSeriesRoots.length}`, { id: 'export-series-pdf' })
      const result = await exportSeriesPdfs(selectedSeriesRoots)

      result.failed.forEach((failure) => {
        toast.error(`《${failure.title || '未命名系列'}》导出失败：${failure.message}`)
      })

      toast.success(`已开始下载 ${result.succeededCount} 份 PDF`, { id: 'export-series-pdf' })
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

      resetBatchSelection()
    } catch (err) {
      console.error('Failed to batch delete blogs:', err)
      toast.error('批量删除失败，请稍后重试')
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
            'flex items-center py-2 px-3 hover:bg-zinc-100 rounded-md cursor-pointer text-sm gap-2 transition-colors',
            isSelected && !isBatchMode ? 'bg-indigo-50 text-indigo-700 font-medium' : 'text-zinc-700'
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
            <BookOpen className={cn('w-4 h-4', isSelected && !isBatchMode ? 'text-indigo-500' : 'text-zinc-400')} />
          )}
          <span className="truncate flex-1 min-w-0" title={node.title || '无标题'}>{node.title || '无标题'}</span>
        </div>

        {hasChildren && isExpanded && (
          <div className="flex flex-col">
            {/* If node is a parent, manually inject the parent node itself as the first "Series Guide" child in the UI */}
            {node.parent_id === null && (
              <div
                className={cn(
                  'flex items-center py-2 px-3 hover:bg-zinc-100 rounded-md cursor-pointer text-sm gap-2 transition-colors',
                  selectedBlog?.id === node.id && !isBatchMode ? 'bg-indigo-50 text-indigo-700 font-medium' : 'text-zinc-700'
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
                <BookOpen className={cn('w-4 h-4', selectedBlog?.id === node.id && !isBatchMode ? 'text-indigo-500' : 'text-zinc-400')} />
                <span className="truncate flex-1 min-w-0" title="系列导读 (概览)">系列导读 (概览)</span>
              </div>
            )}
            {node.children.map(child => renderBlogNode(child, level + 1))}
          </div>
        )}
      </div>
    )
  }

  const headerContent = (
    <div className="p-4 border-b border-zinc-200 flex flex-col gap-4">
      <div className="flex items-center gap-2 font-semibold text-lg text-zinc-800">
        <GitBranch className="w-5 h-5 text-indigo-600" />
        墨言博客助手
      </div>
      <Button
        className="w-full gap-2 bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm"
        onClick={() => {
          if (streamStore.isGenerating || streamStore.isAnalyzing) {
            setShowWorkspaceResetConfirm(true)
          } else {
            streamStore.reset()
            setCurrentView('home-entry')
          }
        }}
      >
        <Plus className="w-4 h-4" />
        新工作区
      </Button>
      <Button
        variant="outline"
        className="w-full gap-2 shadow-sm"
        disabled={isCreatingDraft}
        onClick={async () => {
          if (isCreatingDraft) return
          setIsCreatingDraft(true)
          try {
            await createDraftBlog()
            toast.success('已创建草稿，开始写作吧')
          } catch (err: unknown) {
            const message = err instanceof Error ? err.message : '创建草稿失败'
            toast.error(message)
          } finally {
            setIsCreatingDraft(false)
          }
        }}
      >
        {isCreatingDraft ? <Loader2 className="w-4 h-4 animate-spin" /> : <FilePenLine className="w-4 h-4" />}
        写博客
      </Button>
    </div>
  )

  const footerContent = (
    <div className="p-4 border-t border-zinc-200 flex flex-col gap-2">
      <Button
        variant="ghost"
        className={cn(
          'w-full flex items-center justify-start gap-2',
          currentView === 'knowledge-review' && !selectedBlog ? 'bg-indigo-50 text-indigo-700 font-medium' : 'text-zinc-600 hover:bg-zinc-100 hover:text-zinc-900'
        )}
        onClick={() => {
          setShouldResumeSessionOnOpen(false)
          setCurrentView('knowledge-review')
        }}
      >
        <Sparkles className="w-4 h-4" />
        知识漫游复习
      </Button>
      <Button
        variant="ghost"
        className={cn(
          'w-full flex items-center justify-start gap-2',
          currentView === 'dashboard' && !selectedBlog ? 'bg-indigo-50 text-indigo-700 font-medium' : 'text-zinc-600 hover:bg-zinc-100 hover:text-zinc-900'
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
          setShowLogoutConfirm(true)
        }}
      >
        <LogOut className="w-4 h-4" />
        退出登录
      </Button>
    </div>
  )

  return (
    <>
      <SidebarShell header={headerContent} footer={footerContent}>
        {streamStore.outline && streamStore.outline.length > 0 && (
          <StreamOutlineSection
            outline={streamStore.outline}
            chapterStatus={streamStore.chapterStatus}
            chapterErrors={streamStore.chapterErrors}
            parentBlogId={streamStore.parentBlogId}
            isGenerating={streamStore.isGenerating}
            isAnalyzing={streamStore.isAnalyzing}
            resetStream={streamStore.reset}
            setCurrentView={setCurrentView}
            blogs={blogs}
            fetchBlogs={fetchBlogs}
            selectBlog={(blog) => selectBlog(blog)}
            blogMap={blogMap}
            expandedNodes={expandedNodes}
            setExpandedNodes={(next) => setExpandedNodes(next)}
          />
        )}

        {/* History Blogs Section */}
        <div className="flex-1 flex flex-col min-h-0 overflow-hidden relative">
          <div className="text-xs font-semibold text-zinc-500 uppercase tracking-wider mb-4 flex items-center justify-between shrink-0 px-4 pt-4">
            <span>历史博客</span>
            <div className="flex items-center gap-1">
              <Button
                variant={isBatchMode ? 'secondary' : 'ghost'}
                size="icon"
                className={cn('h-6 w-6', isBatchMode && 'bg-indigo-100 text-indigo-700 hover:bg-indigo-200')}
                onClick={toggleBatchMode}
                title="批量导出"
              >
                <FolderArchive className="w-3.5 h-3.5" />
              </Button>
              <Button variant="ghost" size="icon" className="h-6 w-6" onClick={fetchBlogs} title="刷新列表">
                <RefreshCw className="w-3.5 h-3.5" />
              </Button>
            </div>
          </div>

          <div className={cn('flex-1 overflow-y-auto custom-scrollbar flex flex-col space-y-1 px-4', isBatchMode ? 'pb-24' : 'pb-4')}>
            {blogs.length > 0 ? (
              blogs.map(blog => renderBlogNode(blog))
            ) : (
              <div className="text-sm text-zinc-400 text-center py-6">
                暂无历史记录
              </div>
            )}
          </div>

          {isBatchMode && (
            <SidebarBatchActionBar
              selectedCount={selectedForExport.size}
              canExportSeries={selectedSeriesRoots.length > 0}
              isExporting={isExporting}
              isDeleting={isDeleting}
              isSyncingSeriesToObsidian={isSyncingSeriesToObsidian}
              isExportingPDF={isExportingPDF}
              onCancel={closeBatchMode}
              onExportZip={handleBatchExport}
              onExportSeriesPdf={handleBatchExportSeriesPDF}
              onSyncSeriesToObsidian={handleBatchExportSeriesToObsidian}
              onDelete={handleBatchDeleteClick}
            />
          )}
        </div>
      </SidebarShell>

      <ConfirmDialog
        isOpen={showBatchDeleteConfirm}
        title="确认批量删除"
        message={`确定要删除选中的 ${selectedForExport.size} 篇博客吗？此操作不可恢复。`}
        confirmText="确认删除"
        onConfirm={executeBatchDelete}
        onCancel={() => setShowBatchDeleteConfirm(false)}
        isDestructive={true}
      />

      <ConfirmDialog
        isOpen={showWorkspaceResetConfirm}
        title="开启新工作区"
        message="当前有任务正在执行，确定要终止并开启新工作区吗？"
        confirmText="终止并新建"
        cancelText="取消"
        onConfirm={() => {
          streamStore.reset()
          setCurrentView('home-entry')
          setShowWorkspaceResetConfirm(false)
        }}
        onCancel={() => setShowWorkspaceResetConfirm(false)}
        isDestructive={true}
      />

      <ConfirmDialog
        isOpen={showLogoutConfirm}
        title="退出登录"
        message="确定要退出登录吗？"
        confirmText="退出登录"
        cancelText="取消"
        onConfirm={() => {
          authTokenStore.clearToken()
          setShowLogoutConfirm(false)
        }}
        onCancel={() => setShowLogoutConfirm(false)}
        isDestructive={true}
      />
    </>
  )
}
