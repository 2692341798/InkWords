import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  STREAM_FLUSH_DELAY_MS,
  createChapterChunkBuffer,
  createTextChunkBuffer,
} from './streamFlushBuffer'

describe('streamFlushBuffer', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('merges text chunks only after the flush window elapses', () => {
    vi.useFakeTimers()
    const flushed: string[] = []
    const buffer = createTextChunkBuffer((chunk) => {
      flushed.push(chunk)
    })

    buffer.push('你')
    buffer.push('好')

    expect(flushed).toEqual([])

    vi.advanceTimersByTime(STREAM_FLUSH_DELAY_MS - 1)
    expect(flushed).toEqual([])

    vi.advanceTimersByTime(1)
    expect(flushed).toEqual(['你好'])
  })

  it('drops buffered text when cancelled before flushing', () => {
    vi.useFakeTimers()
    const flushed: string[] = []
    const buffer = createTextChunkBuffer((chunk) => {
      flushed.push(chunk)
    })

    buffer.push('稍后')
    buffer.cancel()
    vi.runAllTimers()

    expect(flushed).toEqual([])
  })

  it('merges chapter chunks by chapter id in a single flush', () => {
    vi.useFakeTimers()
    const flushed: Array<Record<number, string>> = []
    const buffer = createChapterChunkBuffer((updates) => {
      flushed.push(updates)
    })

    buffer.push(2, '第')
    buffer.push(2, '二章')
    buffer.push(3, '第三章')

    expect(flushed).toEqual([])

    vi.advanceTimersByTime(STREAM_FLUSH_DELAY_MS)
    expect(flushed).toEqual([{ 2: '第二章', 3: '第三章' }])
  })
})
