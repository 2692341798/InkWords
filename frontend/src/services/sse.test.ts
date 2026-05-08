import { describe, expect, it } from 'vitest'
import { buildAuthHeader } from './sse'

describe('buildAuthHeader', () => {
  it('returns empty object when token missing', () => {
    expect(buildAuthHeader(null)).toEqual({})
  })

  it('returns Bearer token header when token present', () => {
    expect(buildAuthHeader('t')).toEqual({ Authorization: 'Bearer t' })
  })
})

