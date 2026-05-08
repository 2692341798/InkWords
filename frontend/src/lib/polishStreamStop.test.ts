import { describe, expect, test } from 'vitest'
import { shouldResetPolishState } from './polishStreamStop'

describe('shouldResetPolishState', () => {
  test('done 与 aborted 不需要额外 reset', () => {
    expect(shouldResetPolishState('done')).toBe(false)
    expect(shouldResetPolishState('aborted')).toBe(false)
  })

  test('其它错误需要 reset（避免卡在 loading）', () => {
    expect(shouldResetPolishState('请求失败')).toBe(true)
    expect(shouldResetPolishState('closed by server')).toBe(true)
    expect(shouldResetPolishState('登录已过期，请重新登录')).toBe(true)
  })
})

