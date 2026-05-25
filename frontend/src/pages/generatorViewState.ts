import type { Chapter } from '@/store/streamStore'
import { scenarioModeLabelMap, type ScenarioMode } from '@/lib/scenarioMode'

interface GeneratorViewStateInput {
  outline: Chapter[] | null
  scenarioMode: ScenarioMode
}

/**
 * Why: 创作场景只在生成大纲前可选，避免大纲已生成后再切换场景造成语义歧义。
 */
export function getGeneratorViewState({
  outline,
  scenarioMode,
}: GeneratorViewStateInput) {
  const hasOutline = Boolean(outline && outline.length > 0)

  return {
    shouldShowScenarioSelector: !hasOutline,
    lockedScenarioLabel: hasOutline ? scenarioModeLabelMap[scenarioMode] : null,
  }
}
