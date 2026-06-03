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

  it('tracks chapter quality phase and cache usage for each chapter', () => {
    useStreamStore.getState().setOutline([
      { sort: 1, title: 'Gin 路由', summary: '请求流转' },
    ])

    useStreamStore.getState().updateChapterPhase(1, 'reviewing')
    useStreamStore.getState().setChapterUsage(1, {
      prompt_tokens: 1200,
      completion_tokens: 500,
      prompt_cache_hit_tokens: 900,
      prompt_cache_miss_tokens: 300,
    })

    expect(useStreamStore.getState()).toMatchObject({
      chapterPhases: {
        1: 'reviewing',
      },
      chapterUsage: {
        1: {
          prompt_cache_hit_tokens: 900,
          prompt_cache_miss_tokens: 300,
        },
      },
    })
  })

  it('clears chapter quality phase and cache usage on reset', () => {
    useStreamStore.getState().setOutline([
      { sort: 1, title: 'Gin 路由', summary: '请求流转' },
    ])
    useStreamStore.getState().updateChapterPhase(1, 'streaming')
    useStreamStore.getState().setChapterUsage(1, {
      prompt_tokens: 1200,
      completion_tokens: 500,
      prompt_cache_hit_tokens: 900,
      prompt_cache_miss_tokens: 300,
    })

    useStreamStore.getState().reset()

    expect(useStreamStore.getState()).toMatchObject({
      chapterPhases: {},
      chapterUsage: {},
    })
  })
})
