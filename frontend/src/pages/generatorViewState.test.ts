import { describe, expect, it } from 'vitest'
import { getGeneratorViewState } from './generatorViewState'

describe('getGeneratorViewState', () => {
  it('shows the scenario selector before the outline is generated', () => {
    expect(
      getGeneratorViewState({
        outline: null,
        scenarioMode: 'open_book_exam_review',
      }),
    ).toEqual({
      shouldShowScenarioSelector: true,
      lockedScenarioLabel: null,
    })
  })

  it('hides the scenario selector and exposes the locked scenario label after outline generation', () => {
    expect(
      getGeneratorViewState({
        outline: [{ sort: 1, title: '章节一', summary: '摘要' }],
        scenarioMode: 'open_book_exam_review',
      }),
    ).toEqual({
      shouldShowScenarioSelector: false,
      lockedScenarioLabel: '开卷复习',
    })
  })
})
