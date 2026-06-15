import type { ReactNode } from 'react'
import { Button } from '@/components/ui/button'
import { Panel, SectionHeader, StatusPill } from '@/components/ui/workspace'

interface GeneratorOutlineStageProps {
  lockedScenarioLabel?: string | null
  lockedPromptProfileLabel?: string | null
  outlineEditor: ReactNode
  progressPanel?: ReactNode
  onBack: () => void
}

/**
 * Why: 大纲出现后，用户应只专注于修改结构和开始生成，而不是回看前序配置面板。
 */
export function GeneratorOutlineStage({
  lockedScenarioLabel,
  lockedPromptProfileLabel,
  outlineEditor,
  progressPanel,
  onBack,
}: GeneratorOutlineStageProps) {
  return (
    <Panel className="space-y-5 p-6">
      <SectionHeader
        eyebrow="大纲确认"
        title="确认并调整大纲"
        description="当前页面只保留结构调整和开始生成，前序配置被收进摘要标签。"
        action={<Button variant="outline" onClick={onBack}>返回上一步</Button>}
      />

      {(lockedScenarioLabel || lockedPromptProfileLabel) ? (
        <div className="flex flex-wrap gap-2">
          {lockedScenarioLabel ? <StatusPill>创作场景：{lockedScenarioLabel}</StatusPill> : null}
          {lockedPromptProfileLabel ? <StatusPill>提示词类型：{lockedPromptProfileLabel}</StatusPill> : null}
        </div>
      ) : null}

      {outlineEditor}
      {progressPanel}
    </Panel>
  )
}
