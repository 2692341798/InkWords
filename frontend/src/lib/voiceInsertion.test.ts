import { describe, expect, test } from 'vitest'
import { replaceVoiceSegment } from './voiceInsertion'

describe('replaceVoiceSegment', () => {
  test('插入到锚点位置（无选区）', () => {
    const res = replaceVoiceSegment({
      base: 'hello world',
      anchorStart: 6,
      anchorEnd: 6,
      lastInsertedLength: 0,
      nextText: '语音',
    })
    expect(res.merged).toBe('hello 语音world')
    expect(res.newInsertedLength).toBe(2)
  })

  test('替换上一次插入的区间，避免重复堆叠', () => {
    const first = replaceVoiceSegment({
      base: 'hello world',
      anchorStart: 6,
      anchorEnd: 6,
      lastInsertedLength: 0,
      nextText: 'a',
    })
    expect(first.merged).toBe('hello aworld')

    const second = replaceVoiceSegment({
      base: first.merged,
      anchorStart: 6,
      anchorEnd: 6,
      lastInsertedLength: first.newInsertedLength,
      nextText: 'ab',
    })
    expect(second.merged).toBe('hello abworld')
  })

  test('存在选区时，插入应覆盖选区与上一次插入区间（取更大者）', () => {
    const first = replaceVoiceSegment({
      base: 'hello WORLD !!!',
      anchorStart: 6,
      anchorEnd: 11,
      lastInsertedLength: 0,
      nextText: '语音',
    })
    expect(first.merged).toBe('hello 语音 !!!')

    const second = replaceVoiceSegment({
      base: first.merged,
      anchorStart: 6,
      anchorEnd: 11,
      lastInsertedLength: 2,
      nextText: '语音输入',
    })
    expect(second.merged).toBe('hello 语音输入 !!!')
  })
})

