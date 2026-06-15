import type { ChangeEvent, DragEvent, RefObject } from 'react'
import { Button } from '@/components/ui/button'
import { GitBranch, UploadCloud, Loader2 } from 'lucide-react'
import { useStreamStore } from '@/store/streamStore'
import { useShallow } from 'zustand/react/shallow'

interface GeneratorInputProps {
  gitUrl: string
  setGitUrl: (url: string) => void
  isDragging: boolean
  handleScan: () => void
  handleDragOver: (e: DragEvent<HTMLDivElement>) => void
  handleDragLeave: (e: DragEvent<HTMLDivElement>) => void
  handleDrop: (e: DragEvent<HTMLDivElement>) => void
  handleFileChange: (e: ChangeEvent<HTMLInputElement>) => void
  fileInputRef: RefObject<HTMLInputElement | null>
  stopAnalyzing: () => void
}

export function GeneratorInput({
  gitUrl,
  setGitUrl,
  isDragging,
  handleScan,
  handleDragOver,
  handleDragLeave,
  handleDrop,
  handleFileChange,
  fileInputRef,
  stopAnalyzing
}: GeneratorInputProps) {
  const store = useStreamStore(
    useShallow((state) => ({
      isScanning: state.isScanning,
      isAnalyzing: state.isAnalyzing,
      isGenerating: state.isGenerating,
    })),
  )

  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
      {/* Git URL Input */}
      <div className="choice-tile flex min-h-[220px] flex-col justify-between">
        <div>
        <div className="mb-4 flex h-10 w-10 items-center justify-center rounded-lg bg-[var(--brand-soft)] text-[var(--brand)]">
          <GitBranch size={24} />
        </div>
        <h2 className="mb-2 text-base font-semibold text-foreground">解析开源项目</h2>
        <p className="mb-5 text-sm leading-6 text-muted-foreground">
          输入 GitHub 仓库地址，自动解析代码结构并生成系列教程
        </p>
        </div>
        <div className="w-full flex gap-2">
          <input
            type="text"
            placeholder="https://github.com/user/repo"
            className="min-w-0 flex-1 rounded-lg border border-border bg-secondary/35 px-3 py-2 text-sm text-foreground outline-none transition focus:border-[var(--brand)] focus:bg-card focus:ring-2 focus:ring-ring/40"
            value={gitUrl}
            onChange={(e) => setGitUrl(e.target.value)}
            disabled={store.isScanning || store.isAnalyzing || store.isGenerating}
          />
          {store.isScanning || store.isAnalyzing ? (
            <Button
              variant="destructive"
              onClick={stopAnalyzing}
            >
              取消解析
            </Button>
          ) : (
            <Button
              onClick={handleScan}
              disabled={!gitUrl || store.isGenerating}
            >
              {store.isScanning ? <Loader2 className="w-4 h-4 animate-spin" /> : '扫描目录'}
            </Button>
          )}
        </div>
      </div>

      {/* File Upload */}
      <div
        className={`choice-tile flex min-h-[220px] cursor-pointer flex-col justify-center border-dashed transition-colors
          ${isDragging
            ? 'choice-tile-active'
            : 'choice-tile-muted'
          }
          ${(store.isScanning || store.isAnalyzing || store.isGenerating) ? 'opacity-50 cursor-not-allowed' : ''}
        `}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        onClick={() => {
          if (!store.isScanning && !store.isAnalyzing && !store.isGenerating) {
            fileInputRef.current?.click()
          }
        }}
      >
        <input
          type="file"
          className="hidden"
          ref={fileInputRef}
          onChange={handleFileChange}
          accept=".pdf,.docx,.md,.markdown,.txt,.zip"
          disabled={store.isScanning || store.isAnalyzing || store.isGenerating}
        />
        <div className="mb-4 flex h-10 w-10 items-center justify-center rounded-lg bg-[var(--success-soft)] text-[var(--success)]">
          <UploadCloud size={24} />
        </div>
        <h2 className="mb-2 text-base font-semibold text-foreground">解析本地文档</h2>
        <p className="mb-2 text-sm leading-6 text-muted-foreground">
          拖拽 PDF、DOCX、Markdown、TXT 或 ZIP 课件包到这里
        </p>
        <p className="text-xs text-muted-foreground">
          支持解析生成单篇或多篇系列博客
        </p>
      </div>
    </div>
  )
}
