import { Button } from '../ui/button'
import { Download, FileArchive, FileDown, Loader2, Mic, MicOff, Save, Sparkles, ChevronDown, Wand2 } from 'lucide-react'
import type { BlogNode } from '@/store/blogStore'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../ui/dropdown-menu"

type EditorHeaderProps = {
  selectedBlog: BlogNode
  title: string
  onTitleChange: (value: string) => void
  isSaving: boolean
  lastSaved: Date | null
  isVoiceListening: boolean
  isContinuing: boolean
  isPolishing: boolean
  onToggleVoiceInput: () => void
  onStartPolish: () => void
  onContinueGenerating: () => void
  onExportToObsidian: () => void
  onExportSeriesToObsidian: () => void
  onExportSeriesZip: () => void
  onExportMarkdown: () => void
  onExportPDF: () => void
}

export function EditorHeader(props: EditorHeaderProps) {
  const {
    selectedBlog,
    title,
    onTitleChange,
    isSaving,
    lastSaved,
    isVoiceListening,
    isContinuing,
    isPolishing,
    onToggleVoiceInput,
    onStartPolish,
    onContinueGenerating,
    onExportToObsidian,
    onExportSeriesToObsidian,
    onExportSeriesZip,
    onExportMarkdown,
    onExportPDF,
  } = props

  const isSeriesParent =
    selectedBlog?.parent_id === '00000000-0000-0000-0000-000000000000' &&
    selectedBlog?.children &&
    selectedBlog.children.length > 0

  return (
    <div className="flex h-14 shrink-0 items-center justify-between border-b border-border bg-card px-5 print:hidden">
      <div className="flex items-center gap-4 flex-1">
        <input
          type="text"
          className="w-1/2 border-none bg-transparent text-lg font-semibold text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-0"
          placeholder="输入博客标题..."
          value={title}
          onChange={(e) => onTitleChange(e.target.value)}
        />
        <div className="flex items-center gap-1 text-xs text-muted-foreground">
          {isSaving ? (
            <>
              <Loader2 className="w-3 h-3 animate-spin" /> 保存中...
            </>
          ) : lastSaved ? (
            <>
              <Save className="w-3 h-3" /> 已保存 {lastSaved.toLocaleTimeString()}
            </>
          ) : null}
        </div>
      </div>

      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={onToggleVoiceInput}
          disabled={isContinuing || isPolishing}
          className={
            isVoiceListening
              ? 'gap-1.5 text-red-600 border-red-200 hover:bg-red-50 transition-all duration-200'
              : 'gap-1.5 text-muted-foreground hover:text-foreground transition-all duration-200'
          }
        >
          {isVoiceListening ? <MicOff className="w-4 h-4" /> : <Mic className="w-4 h-4" />}
          {isVoiceListening ? '停止语音' : '语音输入'}
        </Button>

        <Button
          variant="outline"
          size="sm"
          onClick={onStartPolish}
          disabled={isPolishing || isContinuing || isVoiceListening}
          className="gap-1.5 text-muted-foreground hover:text-foreground transition-all duration-200"
        >
          {isPolishing ? <Loader2 className="w-4 h-4 animate-spin" /> : <Wand2 className="w-4 h-4" />}
          {isPolishing ? '润色中...' : '润色'}
        </Button>

        <Button
          variant="outline"
          size="sm"
          onClick={onContinueGenerating}
          disabled={isContinuing || isVoiceListening || isPolishing}
          className="gap-1.5 border-[color-mix(in_srgb,var(--brand)_20%,var(--border))] text-[var(--brand)] hover:bg-[var(--brand-soft)] transition-all duration-200"
        >
          {isContinuing ? <Loader2 className="w-4 h-4 animate-spin" /> : <Sparkles className="w-4 h-4" />}
          继续生成
        </Button>

        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                variant="outline"
                size="sm"
                className="gap-1.5 text-muted-foreground hover:text-foreground transition-all duration-200"
              >
                <Download className="w-4 h-4" />
                导出 / 同步
                <ChevronDown className="w-3 h-3 opacity-50" />
              </Button>
            }
          />
          <DropdownMenuContent align="end" className="w-56 rounded-xl border-border p-1 shadow-lg">
            <DropdownMenuItem onClick={onExportToObsidian} className="gap-2 cursor-pointer rounded-lg py-2 focus:bg-secondary">
              <div className="rounded-md bg-[var(--brand-soft)] p-1.5 text-[var(--brand)]">
                <Sparkles className="w-3.5 h-3.5" />
              </div>
              <div className="flex flex-col">
                <span className="font-medium text-foreground">同步单篇到 Obsidian</span>
                <span className="text-xs text-muted-foreground">直通本地第二大脑</span>
              </div>
            </DropdownMenuItem>

            {isSeriesParent && (
              <DropdownMenuItem
                onClick={onExportSeriesToObsidian}
                className="mt-1 gap-2 cursor-pointer rounded-lg py-2 focus:bg-secondary"
              >
                <div className="rounded-md bg-[var(--brand-soft)] p-1.5 text-[var(--brand)]">
                  <FileArchive className="w-3.5 h-3.5" />
                </div>
                <div className="flex flex-col">
                  <span className="font-medium text-foreground">同步整个系列到 Obsidian</span>
                  <span className="text-xs text-muted-foreground">自动构建双链知识网络</span>
                </div>
              </DropdownMenuItem>
            )}

            <DropdownMenuSeparator className="my-1 bg-border" />

            {isSeriesParent && (
              <DropdownMenuItem onClick={onExportSeriesZip} className="gap-2 cursor-pointer rounded-lg focus:bg-secondary">
                <FileArchive className="w-4 h-4 text-muted-foreground" />
                <span>导出系列 ZIP</span>
              </DropdownMenuItem>
            )}
            <DropdownMenuItem onClick={onExportMarkdown} className="gap-2 cursor-pointer rounded-lg focus:bg-secondary">
              <FileDown className="w-4 h-4 text-muted-foreground" />
              <span>导出为 Markdown</span>
            </DropdownMenuItem>
            <DropdownMenuItem onClick={onExportPDF} className="gap-2 cursor-pointer rounded-lg focus:bg-secondary">
              <Download className="w-4 h-4 text-muted-foreground" />
              <span>打印为 PDF</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  )
}
