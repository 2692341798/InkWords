import { BookOpen, CheckSquare, ChevronDown, ChevronRight, FolderArchive, RefreshCw, Square } from 'lucide-react'
import type { BlogNode } from '@/store/blogStore'
import { cn } from '@/lib/utils'
import { Button } from '../ui/button'
import { SidebarBatchActionBar } from './SidebarBatchActionBar'

interface BlogTreeDisplayProps {
  blogs: BlogNode[]
  expandedNodes: Set<string>
  toggleNode: (id: string) => void
  selectedBlog: BlogNode | null
  isBatchMode: boolean
  selectedForExport: Set<string>
  toggleNodeSelection: (node: BlogNode) => void
  selectBlog: (blog: BlogNode) => void
  fetchBlogs: () => Promise<void>
  toggleBatchMode: () => void
  closeBatchMode: () => void
  isExporting: boolean
  isDeleting: boolean
  isSyncingSeriesToObsidian: boolean
  isExportingPDF: boolean
  onExportZip: () => void
  onExportSeriesPdf: () => void
  onSyncSeriesToObsidian: () => void
  onDelete: () => void
  selectedSeriesRoots: BlogNode[]
}

export function BlogTreeDisplay({
  blogs,
  expandedNodes,
  toggleNode,
  selectedBlog,
  isBatchMode,
  selectedForExport,
  toggleNodeSelection,
  selectBlog,
  fetchBlogs,
  toggleBatchMode,
  closeBatchMode,
  isExporting,
  isDeleting,
  isSyncingSeriesToObsidian,
  isExportingPDF,
  onExportZip,
  onExportSeriesPdf,
  onSyncSeriesToObsidian,
  onDelete,
  selectedSeriesRoots,
}: BlogTreeDisplayProps) {
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

  return (
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
          onExportZip={onExportZip}
          onExportSeriesPdf={onExportSeriesPdf}
          onSyncSeriesToObsidian={onSyncSeriesToObsidian}
          onDelete={onDelete}
        />
      )}
    </div>
  )
}
