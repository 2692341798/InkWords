import { beforeEach, describe, expect, it } from 'vitest'
import { useStreamStore } from './streamStore'

describe('useStreamStore scenario mode', () => {
  beforeEach(() => {
    useStreamStore.getState().reset()
  })

  it('defaults to ebook interpretation and restores it on reset', () => {
    expect(useStreamStore.getState().scenarioMode).toBe('ebook_interpretation')

    useStreamStore.getState().setScenarioMode('beginner_walkthrough')
    useStreamStore.getState().reset()

    expect(useStreamStore.getState().scenarioMode).toBe('ebook_interpretation')
  })

  it('switches to the recommended mode when the source type changes', () => {
    useStreamStore.getState().setSource('git', '', 'https://github.com/inkwords/demo')
    expect(useStreamStore.getState().scenarioMode).toBe('beginner_walkthrough')

    useStreamStore.getState().setSource('file', 'document content')
    expect(useStreamStore.getState().scenarioMode).toBe('ebook_interpretation')
  })

  it('keeps the manual selection when the same source type refreshes', () => {
    useStreamStore.getState().setSource('git', '', 'https://github.com/inkwords/demo')
    useStreamStore.getState().setScenarioMode('open_book_exam_review')

    useStreamStore.getState().setSource('git', 'repo summary', 'https://github.com/inkwords/demo')

    expect(useStreamStore.getState().scenarioMode).toBe('open_book_exam_review')
  })
})
