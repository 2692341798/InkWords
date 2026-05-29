import type { ChangeEvent, DragEvent, RefObject } from 'react'
import { GeneratorInput } from '@/components/generator/GeneratorInput'

interface GeneratorSourceStageProps {
  gitUrl: string
  setGitUrl: (value: string) => void
  isDragging: boolean
  handleScan: () => void
  handleDragOver: (event: DragEvent<HTMLDivElement>) => void
  handleDragLeave: (event: DragEvent<HTMLDivElement>) => void
  handleDrop: (event: DragEvent<HTMLDivElement>) => void
  handleFileChange: (event: ChangeEvent<HTMLInputElement>) => void
  fileInputRef: RefObject<HTMLInputElement | null>
  stopAnalyzing: () => void
}

/**
 * Why: 用户的第一个问题只应该是“资料从哪里来”，所以来源阶段独占主工作区。
 */
export function GeneratorSourceStage(props: GeneratorSourceStageProps) {
  return (
    <section className="rounded-3xl border border-zinc-200 bg-white p-8 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
      <div className="mb-8 space-y-2">
        <h2 className="text-2xl font-semibold text-zinc-900 dark:text-zinc-100">先选择资料来源</h2>
        <p className="text-sm leading-6 text-zinc-500 dark:text-zinc-400">
          先确认你要解析的是 GitHub 仓库还是本地文档，完成这一步后再进入下一页配置解析策略。
        </p>
      </div>
      <GeneratorInput {...props} />
    </section>
  )
}
