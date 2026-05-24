import { describe, expect, it } from 'vitest'
import {
  defaultScenarioModeForSource,
  scenarioModeLabelMap,
  scenarioModeOptions,
} from './scenarioMode'

describe('scenarioMode', () => {
  it('returns beginner walkthrough for git sources', () => {
    expect(defaultScenarioModeForSource('git')).toBe('beginner_walkthrough')
  })

  it('returns ebook interpretation for file and empty sources', () => {
    expect(defaultScenarioModeForSource('file')).toBe('ebook_interpretation')
    expect(defaultScenarioModeForSource(null)).toBe('ebook_interpretation')
  })

  it('exposes Chinese labels and descriptions for the selector', () => {
    expect(scenarioModeLabelMap.open_book_exam_review).toBe('开卷复习')
    expect(scenarioModeOptions).toEqual([
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
    ])
  })
})
