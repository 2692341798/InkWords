import type { ChangeEvent, DragEvent, RefObject } from 'react'
import { Button } from '@/components/ui/button'
import { GitBranch, UploadCloud, Loader2 } from 'lucide-react'
import { useStreamStore } from '@/store/streamStore'

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
  const store = useStreamStore()

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-8 mb-12">
      {/* Git URL Input */}
      <div className="bg-white dark:bg-zinc-900 rounded-xl shadow-sm border border-zinc-200 dark:border-zinc-800 p-6 flex flex-col items-center justify-center">
        <div className="w-12 h-12 bg-blue-50 dark:bg-blue-900/20 text-blue-600 dark:text-blue-400 rounded-full flex items-center justify-center mb-4">
          <GitBranch size={24} />
        </div>
        <h2 className="text-lg font-medium text-zinc-900 dark:text-zinc-100 mb-2">解析开源项目</h2>
        <p className="text-sm text-zinc-500 dark:text-zinc-400 text-center mb-6">
          输入 GitHub 仓库地址，自动解析代码结构并生成系列教程
        </p>
        <div className="w-full flex gap-2">
          <input
            type="text"
            placeholder="https://github.com/user/repo"
            className="flex-1 bg-zinc-50 dark:bg-zinc-800/50 border border-zinc-200 dark:border-zinc-700 rounded-lg px-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400 focus:border-transparent dark:text-zinc-100"
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
        className={`bg-white dark:bg-zinc-900 rounded-xl shadow-sm border-2 border-dashed p-6 flex flex-col items-center justify-center transition-colors cursor-pointer relative overflow-hidden
          ${isDragging
            ? 'border-blue-500 bg-blue-50/50 dark:border-blue-400 dark:bg-blue-900/10'
            : 'border-zinc-200 dark:border-zinc-800 hover:border-zinc-300 dark:hover:border-zinc-700 hover:bg-zinc-50 dark:hover:bg-zinc-800/50'
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
          accept=".pdf,.doc,.docx,.md"
          disabled={store.isScanning || store.isAnalyzing || store.isGenerating}
        />
        <div className="w-12 h-12 bg-emerald-50 dark:bg-emerald-900/20 text-emerald-600 dark:text-emerald-400 rounded-full flex items-center justify-center mb-4">
          <UploadCloud size={24} />
        </div>
        <h2 className="text-lg font-medium text-zinc-900 dark:text-zinc-100 mb-2">解析本地文档</h2>
        <p className="text-sm text-zinc-500 dark:text-zinc-400 text-center mb-2">
          拖拽 PDF, Word, 或 Markdown 文件到这里
        </p>
        <p className="text-xs text-zinc-400 dark:text-zinc-500">
          支持解析生成单篇或多篇系列博客
        </p>
      </div>
    </div>
  )
}