export type ScenarioMode =
  | 'ebook_interpretation'
  | 'open_book_exam_review'
  | 'beginner_walkthrough'

export const scenarioModeLabelMap: Record<ScenarioMode, string> = {
  ebook_interpretation: '电子书解读',
  open_book_exam_review: '开卷复习',
  beginner_walkthrough: '小白教程',
}

export const scenarioModeOptions: Array<{
  value: ScenarioMode
  label: string
  description: string
}> = [
  {
    value: 'ebook_interpretation',
    label: '电子书解读',
    description: '适合章节拆解、观点提炼和连续阅读的内容整理。',
  },
  {
    value: 'open_book_exam_review',
    label: '开卷复习',
    description: '适合课件、题库和知识点速查，强调步骤、清单和答题模板。',
  },
  {
    value: 'beginner_walkthrough',
    label: '小白教程',
    description: '适合开源项目与实战教程，突出环境准备、主链路和排错说明。',
  },
]

export function defaultScenarioModeForSource(
  sourceType: 'git' | 'file' | null,
): ScenarioMode {
  return sourceType === 'git' ? 'beginner_walkthrough' : 'ebook_interpretation'
}
