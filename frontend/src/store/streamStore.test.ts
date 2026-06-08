import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useStreamStore } from './streamStore'
import { STREAM_FLUSH_DELAY_MS } from '../lib/streamFlushBuffer'

describe('useStreamStore scenario mode', () => {
  beforeEach(() => {
    useStreamStore.getState().reset()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('defaults to ebook interpretation and restores it on reset', () => {
    expect(useStreamStore.getState().scenarioMode).toBe('ebook_interpretation')

    useStreamStore.getState().setScenarioMode('beginner_walkthrough')
    useStreamStore.getState().reset()

    expect(useStreamStore.getState().scenarioMode).toBe('ebook_interpretation')
  })

  it('keeps the manual selection even when the source type changes', () => {
    useStreamStore.getState().setScenarioMode('open_book_exam_review')

    useStreamStore.getState().setSource('git', '', 'https://github.com/inkwords/demo')
    expect(useStreamStore.getState().scenarioMode).toBe('open_book_exam_review')

    useStreamStore.getState().setSource('file', 'document content')
    expect(useStreamStore.getState().scenarioMode).toBe('open_book_exam_review')
  })

  it('keeps the manual selection when the same source type refreshes', () => {
    useStreamStore.getState().setSource('git', '', 'https://github.com/inkwords/demo')
    useStreamStore.getState().setScenarioMode('open_book_exam_review')

    useStreamStore.getState().setSource('git', 'repo summary', 'https://github.com/inkwords/demo')

    expect(useStreamStore.getState().scenarioMode).toBe('open_book_exam_review')
  })

  it('stores the resolved prompt profile and clears it on reset', () => {
    useStreamStore.getState().setResolvedPromptProfile(
      {
        key: 'psychology_communication_book',
        displayName: '心理学经典解读',
        documentKind: 'psychology_communication',
        reason: '命中沟通与情绪表达主题',
      },
      'resolved',
    )

    expect(useStreamStore.getState()).toMatchObject({
      classificationStatus: 'resolved',
      classificationReason: '命中沟通与情绪表达主题',
      resolvedPromptProfile: {
        displayName: '心理学经典解读',
      },
    })

    useStreamStore.getState().reset()

    expect(useStreamStore.getState()).toMatchObject({
      classificationStatus: 'idle',
      classificationReason: '',
      resolvedPromptProfile: null,
    })
  })

  it('buffers chapter content until the flush window ends', () => {
    vi.useFakeTimers()
    useStreamStore.getState().setOutline([
      { sort: 1, title: '第一章', summary: '摘要' },
    ])

    useStreamStore.getState().bufferChapterContent(1, 'Hello')
    useStreamStore.getState().bufferChapterContent(1, ' World')

    expect(useStreamStore.getState().chapterContents[1]).toBe('')

    vi.advanceTimersByTime(STREAM_FLUSH_DELAY_MS)

    expect(useStreamStore.getState().chapterContents[1]).toBe('Hello World')
  })

})
