import type { ChangeEvent, DragEvent, RefObject } from 'react'
import { GeneratorInput } from '@/components/generator/GeneratorInput'
import { Panel, SectionHeader } from '@/components/ui/workspace'

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
    <Panel className="p-6">
      <SectionHeader
        title="先选择资料来源"
        description="确认你要解析的是 GitHub 仓库还是本地文档，完成后再进入配置解析策略。"
      />
      <div className="mt-6">
      <GeneratorInput {...props} />
      </div>
    </Panel>
  )
}
