import { useEffect, useMemo, useState } from 'react'
import { useStreamStore } from '@/store/streamStore'
import { useBlogStore } from '@/store/blogStore'
import { useReviewStore } from '@/store/reviewStore'
import type { BlogNode } from '@/store/blogStore'
import { FilePenLine, GitBranch, Loader2, LogOut, Plus, Sparkles, User } from 'lucide-react'
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
import { SidebarShell } from './sidebar/SidebarShell'
import { StreamOutlineSection } from './sidebar/StreamOutlineSection'
import { BlogTreeDisplay } from './sidebar/BlogTreeDisplay'

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
      toast.loading('正在创建 PDF 导出任务...', { id: 'export-series-pdf' })
      const result = await exportSeriesPdfs(selectedSeriesRoots)

      result.failed.forEach((failure) => {
        toast.error(`《${failure.title || '未命名系列'}》导出失败：${failure.message}`)
      })

      if (result.succeededCount > 0) {
        toast.success(`PDF 已生成，开始下载（成功 ${result.succeededCount} 个）`, {
          id: 'export-series-pdf',
        })
      } else {
        toast.error('PDF 导出失败', { id: 'export-series-pdf' })
      }
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

  const headerContent = (
    <div className="border-b border-sidebar-border p-4">
      <div className="flex items-center gap-3">
        <div className="grid h-9 w-9 place-items-center rounded-lg bg-foreground text-sm font-semibold text-background">
          墨
        </div>
        <div className="min-w-0">
          <div className="truncate text-sm font-semibold text-sidebar-foreground">墨言博客助手</div>
          <div className="mt-0.5 text-xs text-muted-foreground">知识写作工作台</div>
        </div>
      </div>

      <div className="mt-4 grid grid-cols-2 gap-2">
        <Button
          className="gap-2"
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
          新建
        </Button>
        <Button
          variant="outline"
          className="gap-2"
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
          写作
        </Button>
      </div>

      <nav className="mt-4 space-y-1">
        <Button
          variant="ghost"
          className={cn(
            'w-full justify-start gap-2',
            currentView === 'home-entry' && !selectedBlog
              ? 'bg-sidebar-accent text-sidebar-foreground font-medium'
              : 'text-muted-foreground hover:text-sidebar-foreground',
          )}
          onClick={() => setCurrentView('home-entry')}
        >
          <GitBranch className="w-4 h-4" />
          工作入口
        </Button>
        <Button
          variant="ghost"
          className={cn(
            'w-full justify-start gap-2',
            currentView === 'knowledge-review' && !selectedBlog
              ? 'bg-sidebar-accent text-sidebar-foreground font-medium'
              : 'text-muted-foreground hover:text-sidebar-foreground',
          )}
          onClick={() => {
            setShouldResumeSessionOnOpen(false)
            setCurrentView('knowledge-review')
          }}
        >
          <Sparkles className="w-4 h-4" />
          知识复习
        </Button>
        <Button
          variant="ghost"
          className={cn(
            'w-full justify-start gap-2',
            currentView === 'dashboard' && !selectedBlog
              ? 'bg-sidebar-accent text-sidebar-foreground font-medium'
              : 'text-muted-foreground hover:text-sidebar-foreground',
          )}
          onClick={() => setCurrentView('dashboard')}
        >
          <User className="w-4 h-4" />
          个人中心
        </Button>
      </nav>
    </div>
  )

  const footerContent = (
    <div className="border-t border-sidebar-border p-4">
      <Button
        variant="ghost"
        className="w-full justify-start gap-2 text-muted-foreground hover:bg-red-50 hover:text-red-600"
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

        <BlogTreeDisplay
          blogs={blogs}
          expandedNodes={expandedNodes}
          toggleNode={toggleNode}
          selectedBlog={selectedBlog}
          isBatchMode={isBatchMode}
          selectedForExport={selectedForExport}
          toggleNodeSelection={toggleNodeSelection}
          selectBlog={(blog) => selectBlog(blog)}
          fetchBlogs={fetchBlogs}
          toggleBatchMode={toggleBatchMode}
          closeBatchMode={closeBatchMode}
          isExporting={isExporting}
          isDeleting={isDeleting}
          isSyncingSeriesToObsidian={isSyncingSeriesToObsidian}
          isExportingPDF={isExportingPDF}
          onExportZip={handleBatchExport}
          onExportSeriesPdf={handleBatchExportSeriesPDF}
          onSyncSeriesToObsidian={handleBatchExportSeriesToObsidian}
          onDelete={handleBatchDeleteClick}
          selectedSeriesRoots={selectedSeriesRoots}
        />
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
