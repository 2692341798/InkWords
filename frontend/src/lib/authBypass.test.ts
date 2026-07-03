import { describe, expect, it } from 'vitest'
import { isAuthBypassEnabled } from './authBypass'

describe('isAuthBypassEnabled', () => {
  it('only enables the development bypass for an explicit true value', () => {
    expect(isAuthBypassEnabled('true')).toBe(true)
    expect(isAuthBypassEnabled('false')).toBe(false)
    expect(isAuthBypassEnabled(undefined)).toBe(false)
  })
})
