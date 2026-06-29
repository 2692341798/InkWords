import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useStreamStore } from '@/store/streamStore'
import { handleSeriesChunkMessage } from './useSeriesGenerator'

describe('handleSeriesChunkMessage', () => {
  beforeEach(() => {
    useStreamStore.getState().reset()
  })

  it('stores quality phases and cache usage from series chunk events', () => {
    useStreamStore.getState().setOutline([
      { sort: 1, title: 'Gin 路由', summary: '请求流转' },
    ])
    useStreamStore.getState().setProgress('准备生成环境...')
    const store = useStreamStore.getState()
    const setProgress = vi.spyOn(store, 'setProgress')

    handleSeriesChunkMessage(
      store,
      JSON.stringify({
        chapter_sort: 1,
        status: 'reviewing',
      }),
    )
    handleSeriesChunkMessage(
      store,
      JSON.stringify({
        chapter_sort: 1,
        status: 'repairing',
      }),
    )
    handleSeriesChunkMessage(
      store,
      JSON.stringify({
        chapter_sort: 1,
        status: 'usage',
        prompt_tokens: 1200,
        completion_tokens: 500,
        prompt_cache_hit_tokens: 900,
        prompt_cache_miss_tokens: 300,
      }),
    )
    handleSeriesChunkMessage(
      store,
      JSON.stringify({
        chapter_sort: 1,
        status: 'streaming',
        content: '终稿片段',
      }),
    )

    expect(useStreamStore.getState()).toMatchObject({
      chapterPhases: {
        1: 'streaming',
      },
      chapterUsage: {
        1: {
          prompt_tokens: 1200,
          prompt_cache_hit_tokens: 900,
        },
      },
      chapterContents: {
        1: '终稿片段',
      },
    })
    expect(setProgress).toHaveBeenCalledWith('')
  })

  it('keeps existing completed status handling for terminal events', () => {
    useStreamStore.getState().setOutline([
      { sort: 1, title: 'Gin 路由', summary: '请求流转' },
    ])
    const store = useStreamStore.getState()

    handleSeriesChunkMessage(
      store,
      JSON.stringify({
        chapter_sort: 1,
        status: 'completed',
      }),
    )

    expect(useStreamStore.getState()).toMatchObject({
      chapterStatus: {
        1: 'completed',
      },
      chapterPhases: {
        1: 'completed',
      },
    })
  })
})
