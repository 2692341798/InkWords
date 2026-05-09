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
    <div className="h-14 border-b border-zinc-200 flex items-center justify-between px-6 shrink-0 print:hidden">
      <div className="flex items-center gap-4 flex-1">
        <input
          type="text"
          className="text-lg font-semibold bg-transparent border-none focus:outline-none focus:ring-0 text-zinc-800 placeholder-zinc-400 w-1/2"
          placeholder="输入博客标题..."
          value={title}
          onChange={(e) => onTitleChange(e.target.value)}
        />
        <div className="text-xs text-zinc-400 flex items-center gap-1">
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

      <div className="flex items-center gap-3">
        <Button
          variant="outline"
          size="sm"
          onClick={onToggleVoiceInput}
          disabled={isContinuing || isPolishing}
          className={
            isVoiceListening
              ? 'gap-1.5 text-red-600 border-red-200 hover:bg-red-50 transition-all duration-200'
              : 'gap-1.5 text-zinc-700 hover:text-zinc-900 transition-all duration-200 shadow-sm'
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
          className="gap-1.5 text-emerald-700 border-emerald-200 hover:bg-emerald-50 transition-all duration-200"
        >
          {isPolishing ? <Loader2 className="w-4 h-4 animate-spin" /> : <Wand2 className="w-4 h-4" />}
          {isPolishing ? '润色中...' : '润色'}
        </Button>

        <Button
          variant="outline"
          size="sm"
          onClick={onContinueGenerating}
          disabled={isContinuing || isVoiceListening || isPolishing}
          className="gap-1.5 text-indigo-600 border-indigo-200 hover:bg-indigo-50 transition-all duration-200"
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
                className="gap-1.5 text-zinc-700 hover:text-zinc-900 transition-all duration-200 shadow-sm"
              >
                <Download className="w-4 h-4" />
                导出 / 同步
                <ChevronDown className="w-3 h-3 opacity-50" />
              </Button>
            }
          />
          <DropdownMenuContent align="end" className="w-56 shadow-xl border-zinc-200/60 rounded-xl p-1">
            <DropdownMenuItem onClick={onExportToObsidian} className="gap-2 cursor-pointer focus:bg-zinc-100 rounded-lg py-2">
              <div className="bg-indigo-50 text-indigo-600 p-1.5 rounded-md">
                <Sparkles className="w-3.5 h-3.5" />
              </div>
              <div className="flex flex-col">
                <span className="font-medium text-zinc-900">同步单篇到 Obsidian</span>
                <span className="text-xs text-zinc-500">直通本地第二大脑</span>
              </div>
            </DropdownMenuItem>

            {isSeriesParent && (
              <DropdownMenuItem
                onClick={onExportSeriesToObsidian}
                className="gap-2 cursor-pointer focus:bg-zinc-100 rounded-lg py-2 mt-1"
              >
                <div className="bg-indigo-50 text-indigo-600 p-1.5 rounded-md">
                  <FileArchive className="w-3.5 h-3.5" />
                </div>
                <div className="flex flex-col">
                  <span className="font-medium text-zinc-900">同步整个系列到 Obsidian</span>
                  <span className="text-xs text-zinc-500">自动构建双链知识网络</span>
                </div>
              </DropdownMenuItem>
            )}

            <DropdownMenuSeparator className="bg-zinc-100 my-1" />

            {isSeriesParent && (
              <DropdownMenuItem onClick={onExportSeriesZip} className="gap-2 cursor-pointer focus:bg-zinc-100 rounded-lg">
                <FileArchive className="w-4 h-4 text-zinc-500" />
                <span>导出系列 ZIP</span>
              </DropdownMenuItem>
            )}
            <DropdownMenuItem onClick={onExportMarkdown} className="gap-2 cursor-pointer focus:bg-zinc-100 rounded-lg">
              <FileDown className="w-4 h-4 text-zinc-500" />
              <span>导出为 Markdown</span>
            </DropdownMenuItem>
            <DropdownMenuItem onClick={onExportPDF} className="gap-2 cursor-pointer focus:bg-zinc-100 rounded-lg">
              <Download className="w-4 h-4 text-zinc-500" />
              <span>打印为 PDF</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  )
}

