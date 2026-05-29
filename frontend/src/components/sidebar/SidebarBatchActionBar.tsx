import { ChevronDown, Download, FileArchive, Loader2, Sparkles, Trash2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

interface SidebarBatchActionBarProps {
  selectedCount: number
  canExportSeries: boolean
  isExporting: boolean
  isDeleting: boolean
  isSyncingSeriesToObsidian: boolean
  isExportingPDF: boolean
  onCancel: () => void
  onExportZip: () => void
  onExportSeriesPdf: () => void
  onSyncSeriesToObsidian: () => void
  onDelete: () => void
}

/**
 * Why: 批量操作区要同时处理“可选动作、忙碌态、禁用态”三类规则，
 * 从 Sidebar 容器抽离后，容器就只保留流程编排，不再掺杂按钮矩阵细节。
 */
export function SidebarBatchActionBar({
  selectedCount,
  canExportSeries,
  isExporting,
  isDeleting,
  isSyncingSeriesToObsidian,
  isExportingPDF,
  onCancel,
  onExportZip,
  onExportSeriesPdf,
  onSyncSeriesToObsidian,
  onDelete,
}: SidebarBatchActionBarProps) {
  const isAnyBusy = isExporting || isDeleting || isSyncingSeriesToObsidian || isExportingPDF
  const isZipDisabled = selectedCount === 0 || isExporting || isDeleting
  const isSeriesDisabled = !canExportSeries || isAnyBusy
  const isDeleteDisabled = selectedCount === 0 || isAnyBusy

  return (
    <div className="absolute bottom-0 left-0 right-0 bg-white border-t border-zinc-200 p-3 shadow-[0_-4px_6px_-1px_rgba(0,0,0,0.05)] flex flex-col gap-2">
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs text-zinc-500">已选 {selectedCount} 项</span>
        <Button variant="ghost" size="sm" onClick={onCancel}>取消</Button>
      </div>
      <div className="flex items-center justify-end gap-2">
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button variant="outline" size="sm" disabled={isAnyBusy}>
                <Download data-icon="inline-start" />
                导出 / 同步
                <ChevronDown data-icon="inline-end" className="opacity-60" />
              </Button>
            }
          />
          <DropdownMenuContent align="end" className="w-56">
            <DropdownMenuItem
              onClick={onExportSeriesPdf}
              disabled={isSeriesDisabled}
              className="cursor-pointer"
            >
              <Download data-icon="inline-start" className="text-muted-foreground" />
              <span>导出系列 PDF</span>
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={onExportZip}
              disabled={isZipDisabled}
              className="cursor-pointer"
            >
              <FileArchive data-icon="inline-start" className="text-muted-foreground" />
              <span>导出 ZIP</span>
            </DropdownMenuItem>
            <DropdownMenuSeparator className="my-1" />
            <DropdownMenuItem
              onClick={onSyncSeriesToObsidian}
              disabled={isSeriesDisabled}
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
          onClick={onDelete}
          disabled={isDeleteDisabled}
          title="批量删除"
        >
          {isDeleting ? <Loader2 className="animate-spin" /> : <Trash2 />}
        </Button>
      </div>
    </div>
  )
}
