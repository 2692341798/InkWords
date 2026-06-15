import type { ReactNode } from 'react'
import { Button } from '@/components/ui/button'
import { Panel, SectionHeader, StatusPill } from '@/components/ui/workspace'

interface GeneratorConfigureStageProps {
  sourceLabel: string
  scenarioSelector: ReactNode
  modulePicker?: ReactNode
  fileSummary?: ReactNode
  progressPanel?: ReactNode
  onBack: () => void
}

/**
 * Why: 选择来源之后，用户只需要处理“如何解析”这一件事，避免继续看到来源入口和大纲编辑。
 */
export function GeneratorConfigureStage({
  sourceLabel,
  scenarioSelector,
  modulePicker,
  fileSummary,
  progressPanel,
  onBack,
}: GeneratorConfigureStageProps) {
  return (
    <Panel className="space-y-5 p-6">
      <SectionHeader
        eyebrow="解析配置"
        title="配置解析方式"
        description="这一步只保留当前来源所需的配置项，确认后系统会生成可编辑大纲。"
        action={
          <div className="flex flex-wrap items-center gap-2">
            <StatusPill>当前来源：{sourceLabel}</StatusPill>
            <Button variant="outline" onClick={onBack}>返回上一步</Button>
          </div>
        }
      />

      {scenarioSelector}
      {modulePicker}
      {fileSummary}
      {progressPanel}
    </Panel>
  )
}
